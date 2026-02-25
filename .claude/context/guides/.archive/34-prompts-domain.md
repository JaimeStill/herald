# 34 - Prompts Domain Implementation

## Problem Context

Herald needs a CRUD domain for named prompt instruction overrides that target specific workflow stages (init/classify/enhance). The prompts domain manages the tunable instruction layer — the output format layer is hard-coded in `workflow/prompts.go` and composed at execution time. An `active` boolean with a partial unique index enforces at most one active prompt per stage, allowing the workflow to resolve instructions as: active prompt for stage > hard-coded default.

## Architecture Approach

Follow the documents domain pattern exactly: System interface → repo struct → Handler with Routes(). The prompts domain is simpler than documents (no blob storage, no file uploads) but adds Update and Activate/Deactivate operations. Stage validation occurs at the application layer before DB operations for clearer error messages. Activation swaps the active prompt atomically within a transaction.

## Implementation

### Step 1: Database Migration

Create migration files to add the `active` column and partial unique index to the existing `prompts` table.

**New file: `cmd/migrate/migrations/000003_prompts_active.up.sql`**

```sql
ALTER TABLE prompts
  ADD COLUMN active BOOLEAN NOT NULL DEFAULT false;

CREATE UNIQUE INDEX idx_prompts_stage_active
  ON prompts(stage)
  WHERE active = true;
```

**New file: `cmd/migrate/migrations/000003_prompts_active.down.sql`**

```sql
DROP INDEX IF EXISTS idx_prompts_stage_active;

ALTER TABLE prompts
  DROP COLUMN IF EXISTS active;
```

### Step 2: Entity and Stage Validation

Remove the placeholder `internal/prompts/doc.go` and create the entity types.

**Delete: `internal/prompts/doc.go`**

**New file: `internal/prompts/prompt.go`**

```go
package prompts

import (
	"encoding/json"
	"slices"

	"github.com/google/uuid"
)

type Stage string

const (
	StageInit     Stage = "init"
	StageClassify Stage = "classify"
	StageEnhance  Stage = "enhance"
)

var stages = []Stage{StageInit, StageClassify, StageEnhance}

func Stages() []Stage {
	return stages
}

func (s *Stage) UnmarshalJSON(data []byte) error {
	var raw string
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	v := Stage(raw)
	if !slices.Contains(stages, v) {
		return ErrInvalidStage
	}
	*s = v
	return nil
}

type Prompt struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	Stage        Stage     `json:"stage"`
	Instructions string    `json:"instructions"`
	Description  *string   `json:"description"`
	Active       bool      `json:"active"`
}

type CreateCommand struct {
	Name         string  `json:"name"`
	Stage        Stage   `json:"stage"`
	Instructions string  `json:"instructions"`
	Description  *string `json:"description"`
}

type UpdateCommand struct {
	Name         string  `json:"name"`
	Stage        Stage   `json:"stage"`
	Instructions string  `json:"instructions"`
	Description  *string `json:"description"`
}
```

### Step 3: Domain Errors

**New file: `internal/prompts/errors.go`**

```go
package prompts

import (
	"errors"
	"net/http"
)

var (
	ErrNotFound     = errors.New("prompt not found")
	ErrDuplicate    = errors.New("prompt name already exists")
	ErrInvalidStage = errors.New("stage must be init, classify, or enhance")
)

func MapHTTPStatus(err error) int {
	if errors.Is(err, ErrNotFound) {
		return http.StatusNotFound
	}
	if errors.Is(err, ErrDuplicate) {
		return http.StatusConflict
	}
	if errors.Is(err, ErrInvalidStage) {
		return http.StatusBadRequest
	}
	return http.StatusInternalServerError
}
```

### Step 4: Data Mapping

**New file: `internal/prompts/mapping.go`**

```go
package prompts

import (
	"net/url"
	"strconv"

	"github.com/JaimeStill/herald/pkg/query"
	"github.com/JaimeStill/herald/pkg/repository"
)

var projection = query.
	NewProjectionMap("public", "prompts", "p").
	Project("id", "ID").
	Project("name", "Name").
	Project("stage", "Stage").
	Project("instructions", "Instructions").
	Project("description", "Description").
	Project("active", "Active")

var defaultSort = query.SortField{
	Field: "Name",
}

type Filters struct {
	Stage  *Stage  `json:"stage,omitempty"`
	Name   *string `json:"name,omitempty"`
	Active *bool   `json:"active,omitempty"`
}

func (f Filters) Apply(b *query.Builder) *query.Builder {
	return b.
		WhereEquals("Stage", f.Stage).
		WhereContains("Name", f.Name).
		WhereEquals("Active", f.Active)
}

func FiltersFromQuery(values url.Values) Filters {
	var f Filters

	if s := values.Get("stage"); s != "" {
		stage := Stage(s)
		f.Stage = &stage
	}

	if n := values.Get("name"); n != "" {
		f.Name = &n
	}

	if a := values.Get("active"); a != "" {
		if v, err := strconv.ParseBool(a); err == nil {
			f.Active = &v
		}
	}

	return f
}

func scanPrompt(s repository.Scanner) (Prompt, error) {
	var p Prompt
	err := s.Scan(
		&p.ID,
		&p.Name,
		&p.Stage,
		&p.Instructions,
		&p.Description,
		&p.Active,
	)
	return p, err
}
```

### Step 5: System Interface

**New file: `internal/prompts/system.go`**

```go
package prompts

import (
	"context"

	"github.com/google/uuid"

	"github.com/JaimeStill/herald/pkg/pagination"
)

type System interface {
	Handler() *Handler

	List(
		ctx context.Context,
		page pagination.PageRequest,
		filters Filters,
	) (*pagination.PageResult[Prompt], error)

	Find(ctx context.Context, id uuid.UUID) (*Prompt, error)
	Create(ctx context.Context, cmd CreateCommand) (*Prompt, error)
	Update(ctx context.Context, id uuid.UUID, cmd UpdateCommand) (*Prompt, error)
	Delete(ctx context.Context, id uuid.UUID) error
	Activate(ctx context.Context, id uuid.UUID) (*Prompt, error)
	Deactivate(ctx context.Context, id uuid.UUID) (*Prompt, error)
}
```

### Step 6: Repository

**New file: `internal/prompts/repository.go`**

```go
package prompts

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/JaimeStill/herald/pkg/pagination"
	"github.com/JaimeStill/herald/pkg/query"
	"github.com/JaimeStill/herald/pkg/repository"
)

type repo struct {
	db         *sql.DB
	logger     *slog.Logger
	pagination pagination.Config
}

func New(
	db *sql.DB,
	logger *slog.Logger,
	pagination pagination.Config,
) System {
	return &repo{
		db:         db,
		logger:     logger.With("system", "prompts"),
		pagination: pagination,
	}
}

func (r *repo) Handler() *Handler {
	return NewHandler(r, r.logger, r.pagination)
}

func (r *repo) List(
	ctx context.Context,
	page pagination.PageRequest,
	filters Filters,
) (*pagination.PageResult[Prompt], error) {
	page.Normalize(r.pagination)

	qb := query.
		NewBuilder(projection, defaultSort).
		WhereSearch(page.Search, "Name", "Description")

	filters.Apply(qb)

	if len(page.Sort) > 0 {
		qb.OrderByFields(page.Sort)
	}

	countSQL, countArgs := qb.BuildCount()
	var total int
	if err := r.db.QueryRowContext(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count prompts: %w", err)
	}

	pageSQL, pageArgs := qb.BuildPage(page.Page, page.PageSize)
	prompts, err := repository.QueryMany(ctx, r.db, pageSQL, pageArgs, scanPrompt)
	if err != nil {
		return nil, fmt.Errorf("query prompts: %w", err)
	}

	result := pagination.NewPageResult(prompts, total, page.Page, page.PageSize)
	return &result, nil
}

func (r *repo) Find(ctx context.Context, id uuid.UUID) (*Prompt, error) {
	q, args := query.NewBuilder(projection).BuildSingle("ID", id)

	p, err := repository.QueryOne(ctx, r.db, q, args, scanPrompt)
	if err != nil {
		return nil, repository.MapError(err, ErrNotFound, ErrDuplicate)
	}
	return &p, nil
}

func (r *repo) Create(ctx context.Context, cmd CreateCommand) (*Prompt, error) {
	q := `
		INSERT INTO prompts(name, stage, instructions, description)
		VALUES ($1, $2, $3, $4)
		RETURNING id, name, stage, instructions, description, active`

	args := []any{cmd.Name, cmd.Stage, cmd.Instructions, cmd.Description}

	p, err := repository.WithTx(ctx, r.db, func(tx *sql.Tx) (Prompt, error) {
		return repository.QueryOne(ctx, tx, q, args, scanPrompt)
	})

	if err != nil {
		return nil, repository.MapError(err, ErrNotFound, ErrDuplicate)
	}

	r.logger.Info("prompt created", "id", p.ID, "name", p.Name, "stage", p.Stage)
	return &p, nil
}

func (r *repo) Update(ctx context.Context, id uuid.UUID, cmd UpdateCommand) (*Prompt, error) {
	q := `
		UPDATE prompts
		SET name = $1, stage = $2, instructions = $3, description = $4
		WHERE id = $5
		RETURNING id, name, stage, instructions, description, active`

	args := []any{cmd.Name, cmd.Stage, cmd.Instructions, cmd.Description, id}

	p, err := repository.WithTx(ctx, r.db, func(tx *sql.Tx) (Prompt, error) {
		return repository.QueryOne(ctx, tx, q, args, scanPrompt)
	})

	if err != nil {
		return nil, repository.MapError(err, ErrNotFound, ErrDuplicate)
	}

	r.logger.Info("prompt updated", "id", p.ID, "name", p.Name)
	return &p, nil
}

func (r *repo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := repository.WithTx(ctx, r.db, func(tx *sql.Tx) (struct{}, error) {
		if err := repository.ExecExpectOne(
			ctx, tx,
			"DELETE FROM prompts WHERE id = $1",
			id,
		); err != nil {
			return struct{}{}, err
		}
		return struct{}{}, nil
	})

	if err != nil {
		return repository.MapError(err, ErrNotFound, ErrDuplicate)
	}

	r.logger.Info("prompt deleted", "id", id)
	return nil
}

func (r *repo) Activate(ctx context.Context, id uuid.UUID) (*Prompt, error) {
	p, err := repository.WithTx(ctx, r.db, func(tx *sql.Tx) (Prompt, error) {
		// Find the prompt to get its stage
		findQ, findArgs := query.NewBuilder(projection).BuildSingle("ID", id)
		target, err := repository.QueryOne(ctx, tx, findQ, findArgs, scanPrompt)
		if err != nil {
			return Prompt{}, err
		}

		// Deactivate current active for this stage (if any)
		_, err = tx.ExecContext(ctx,
			"UPDATE prompts SET active = false WHERE stage = $1 AND active = true",
			target.Stage,
		)
		if err != nil {
			return Prompt{}, fmt.Errorf("deactivate current: %w", err)
		}

		// Activate the target
		activateQ := `
			UPDATE prompts SET active = true
			WHERE id = $1
			RETURNING id, name, stage, instructions, description, active`

		return repository.QueryOne(ctx, tx, activateQ, []any{id}, scanPrompt)
	})

	if err != nil {
		return nil, repository.MapError(err, ErrNotFound, ErrDuplicate)
	}

	r.logger.Info("prompt activated", "id", p.ID, "name", p.Name, "stage", p.Stage)
	return &p, nil
}

func (r *repo) Deactivate(ctx context.Context, id uuid.UUID) (*Prompt, error) {
	q := `
		UPDATE prompts SET active = false
		WHERE id = $1
		RETURNING id, name, stage, instructions, description, active`

	p, err := repository.WithTx(ctx, r.db, func(tx *sql.Tx) (Prompt, error) {
		return repository.QueryOne(ctx, tx, q, []any{id}, scanPrompt)
	})

	if err != nil {
		return nil, repository.MapError(err, ErrNotFound, ErrDuplicate)
	}

	r.logger.Info("prompt deactivated", "id", p.ID, "name", p.Name, "stage", p.Stage)
	return &p, nil
}
```

### Step 7: HTTP Handler

**New file: `internal/prompts/handler.go`**

```go
package prompts

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	"github.com/JaimeStill/herald/pkg/handlers"
	"github.com/JaimeStill/herald/pkg/pagination"
	"github.com/JaimeStill/herald/pkg/routes"
)

type Handler struct {
	sys        System
	logger     *slog.Logger
	pagination pagination.Config
}

type SearchRequest struct {
	pagination.PageRequest
	Filters
}

func NewHandler(
	sys System,
	logger *slog.Logger,
	pagination pagination.Config,
) *Handler {
	return &Handler{
		sys:        sys,
		logger:     logger.With("handler", "prompts"),
		pagination: pagination,
	}
}

func (h *Handler) Routes() routes.Group {
	return routes.Group{
		Prefix: "/prompts",
		Routes: []routes.Route{
			{Method: "GET", Pattern: "", Handler: h.List},
			{Method: "GET", Pattern: "/stages", Handler: h.Stages},
			{Method: "GET", Pattern: "/{id}", Handler: h.Find},
			{Method: "POST", Pattern: "", Handler: h.Create},
			{Method: "PUT", Pattern: "/{id}", Handler: h.Update},
			{Method: "DELETE", Pattern: "/{id}", Handler: h.Delete},
			{Method: "POST", Pattern: "/search", Handler: h.Search},
			{Method: "POST", Pattern: "/{id}/activate", Handler: h.Activate},
			{Method: "POST", Pattern: "/{id}/deactivate", Handler: h.Deactivate},
		},
	}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	page := pagination.PageRequestFromQuery(r.URL.Query(), h.pagination)
	filters := FiltersFromQuery(r.URL.Query())

	result, err := h.sys.List(r.Context(), page, filters)
	if err != nil {
		handlers.RespondError(w, h.logger, http.StatusInternalServerError, err)
		return
	}

	handlers.RespondJSON(w, http.StatusOK, result)
}

func (h *Handler) Stages(w http.ResponseWriter, r *http.Request) {
	handlers.RespondJSON(w, http.StatusOK, Stages())
}

func (h *Handler) Find(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, ErrNotFound)
		return
	}

	prompt, err := h.sys.Find(r.Context(), id)
	if err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(err), err)
		return
	}

	handlers.RespondJSON(w, http.StatusOK, prompt)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var cmd CreateCommand
	if err := json.NewDecoder(r.Body).Decode(&cmd); err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, err)
		return
	}

	prompt, err := h.sys.Create(r.Context(), cmd)
	if err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(err), err)
		return
	}

	handlers.RespondJSON(w, http.StatusCreated, prompt)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, ErrNotFound)
		return
	}

	var cmd UpdateCommand
	if err := json.NewDecoder(r.Body).Decode(&cmd); err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, err)
		return
	}

	prompt, err := h.sys.Update(r.Context(), id, cmd)
	if err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(err), err)
		return
	}

	handlers.RespondJSON(w, http.StatusOK, prompt)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, ErrNotFound)
		return
	}

	if err := h.sys.Delete(r.Context(), id); err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(err), err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	var req SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, err)
		return
	}

	req.PageRequest.Normalize(h.pagination)

	result, err := h.sys.List(r.Context(), req.PageRequest, req.Filters)
	if err != nil {
		handlers.RespondError(w, h.logger, http.StatusInternalServerError, err)
		return
	}

	handlers.RespondJSON(w, http.StatusOK, result)
}

func (h *Handler) Activate(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, ErrNotFound)
		return
	}

	prompt, err := h.sys.Activate(r.Context(), id)
	if err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(err), err)
		return
	}

	handlers.RespondJSON(w, http.StatusOK, prompt)
}

func (h *Handler) Deactivate(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, ErrNotFound)
		return
	}

	prompt, err := h.sys.Deactivate(r.Context(), id)
	if err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(err), err)
		return
	}

	handlers.RespondJSON(w, http.StatusOK, prompt)
}
```

### Step 8: Wire into API Domain and Routes

**Modify: `internal/api/domain.go`**

Add the prompts import and system field:

```go
import (
	"github.com/JaimeStill/herald/internal/documents"
	"github.com/JaimeStill/herald/internal/prompts"
)

type Domain struct {
	Documents documents.System
	Prompts   prompts.System
}

func NewDomain(runtime *Runtime) *Domain {
	docsSystem := documents.New(
		runtime.Database.Connection(),
		runtime.Storage,
		runtime.Logger,
		runtime.Pagination,
	)

	promptsSystem := prompts.New(
		runtime.Database.Connection(),
		runtime.Logger,
		runtime.Pagination,
	)

	return &Domain{
		Documents: docsSystem,
		Prompts:   promptsSystem,
	}
}
```

**Modify: `internal/api/routes.go`**

Add the prompts routes registration:

```go
func registerRoutes(
	mux *http.ServeMux,
	domain *Domain,
	cfg *config.Config,
	runtime *Runtime,
) {
	documentsRoutes := domain.Documents.Handler(cfg.API.MaxUploadSizeBytes()).Routes()
	storageRoutes := newStorageHandler(
		runtime.Storage,
		runtime.Logger,
		cfg.Storage.MaxListSize,
	).routes()
	promptsRoutes := domain.Prompts.Handler().Routes()

	routes.Register(
		mux,
		documentsRoutes,
		storageRoutes,
		promptsRoutes,
	)
}
```

## Validation Criteria

- [ ] Migration `000003_prompts_active` applies cleanly
- [ ] All 9 endpoints operational (List, Stages, Find, Create, Update, Delete, Search, Activate, Deactivate)
- [ ] Stage validation rejects invalid stages with 400 response
- [ ] Unique name constraint produces 409 response
- [ ] Partial unique index enforces at most one active prompt per stage
- [ ] Activate atomically swaps active prompt within a transaction
- [ ] Deactivate clears active flag (stage falls back to hard-coded defaults)
- [ ] Pagination, search, and filtering work correctly (including active filter)
- [ ] Domain wired into api.Domain and routes registered
- [ ] `go vet ./...` passes
- [ ] `go mod tidy` produces no changes
- [ ] Server builds and starts: `mise run build && mise run dev`
