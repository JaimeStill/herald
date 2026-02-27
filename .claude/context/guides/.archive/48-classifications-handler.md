# 48 - Classifications Handler, API Wiring, and API Cartographer Docs

## Problem Context

The classifications domain's persistence layer (System, Repository, types, mapping, errors) was implemented in #47. This issue builds the HTTP layer on top: a Handler struct with 8 endpoints, wiring into the API module, and API Cartographer documentation. After this, the full classifications vertical is operational end-to-end.

## Architecture Approach

Follow the established handler pattern from `internal/documents/handler.go` and `internal/prompts/handler.go`. The Handler struct holds a System reference, logger, and pagination config. Routes are defined via `routes.Group`.

**Layered runtime convention**: Workflow is a sub-dependency of classifications, not a peer. The `classifications.New` constructor receives raw infrastructure dependencies and peer systems, then internally constructs the `workflow.Runtime`. This keeps workflow composition encapsulated within classifications and removes the workflow import from `api/domain.go`. The workflow runtime receives `logger.With("workflow", "classify")` to differentiate workflow operations from classifications system operations in logs.

## Implementation

### Step 1: Refactor `classifications.New` to internalize workflow runtime

**`internal/classifications/repository.go`** — change `New` to accept raw dependencies and construct `workflow.Runtime` internally:

```go
func New(
	db *sql.DB,
	agent gaconfig.AgentConfig,
	logger *slog.Logger,
	pagination pagination.Config,
	storage storage.System,
	docs documents.System,
	prompts prompts.System,
) System {
	sysLogger := logger.With("system", "classifications")

	rt := &workflow.Runtime{
		Agent:     agent,
		Storage:   storage,
		Documents: docs,
		Prompts:   prompts,
		Logger:    logger.With("workflow", "classify"),
	}

	return &repo{
		db:         db,
		rt:         rt,
		logger:     sysLogger,
		pagination: pagination,
	}
}
```

Update the `repo` struct imports to add `gaconfig`, `storage`, `documents`, and `prompts`:

```go
import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"slices"

	"github.com/google/uuid"

	"github.com/JaimeStill/herald/internal/documents"
	"github.com/JaimeStill/herald/internal/prompts"
	"github.com/JaimeStill/herald/internal/workflow"
	"github.com/JaimeStill/herald/pkg/pagination"
	"github.com/JaimeStill/herald/pkg/query"
	"github.com/JaimeStill/herald/pkg/repository"
	"github.com/JaimeStill/herald/pkg/storage"

	gaconfig "github.com/JaimeStill/go-agents/pkg/config"
)
```

### Step 2: Add `Handler()` to System interface and repository

**`internal/classifications/system.go`** — add `Handler()` method to the interface:

```go
type System interface {
	Handler() *Handler

	List(
		ctx context.Context,
		page pagination.PageRequest,
		filters Filters,
	) (*pagination.PageResult[Classification], error)

	Find(ctx context.Context, id uuid.UUID) (*Classification, error)
	FindByDocument(ctx context.Context, documentID uuid.UUID) (*Classification, error)
	Classify(ctx context.Context, documentID uuid.UUID) (*Classification, error)
	Validate(ctx context.Context, id uuid.UUID, cmd ValidateCommand) (*Classification, error)
	Update(ctx context.Context, id uuid.UUID, cmd UpdateCommand) (*Classification, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
```

**`internal/classifications/repository.go`** — add `Handler()` implementation on `repo`, placed after the `New` constructor:

```go
func (r *repo) Handler() *Handler {
	return NewHandler(r, r.logger, r.pagination)
}
```

### Step 3: Create `internal/classifications/handler.go`

**New file.** Complete implementation:

```go
package classifications

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
		logger:     logger.With("handler", "classifications"),
		pagination: pagination,
	}
}

func (h *Handler) Routes() routes.Group {
	return routes.Group{
		Prefix: "/classifications",
		Routes: []routes.Route{
			{Method: "GET", Pattern: "", Handler: h.List},
			{Method: "GET", Pattern: "/{id}", Handler: h.Find},
			{Method: "GET", Pattern: "/document/{id}", Handler: h.FindByDocument},
			{Method: "POST", Pattern: "/search", Handler: h.Search},
			{Method: "POST", Pattern: "/{documentId}", Handler: h.Classify},
			{Method: "POST", Pattern: "/{id}/validate", Handler: h.Validate},
			{Method: "PUT", Pattern: "/{id}", Handler: h.Update},
			{Method: "DELETE", Pattern: "/{id}", Handler: h.Delete},
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

func (h *Handler) Find(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, ErrNotFound)
		return
	}

	c, err := h.sys.Find(r.Context(), id)
	if err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(err), err)
		return
	}

	handlers.RespondJSON(w, http.StatusOK, c)
}

func (h *Handler) FindByDocument(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, ErrNotFound)
		return
	}

	c, err := h.sys.FindByDocument(r.Context(), id)
	if err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(err), err)
		return
	}

	handlers.RespondJSON(w, http.StatusOK, c)
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

func (h *Handler) Classify(w http.ResponseWriter, r *http.Request) {
	documentID, err := uuid.Parse(r.PathValue("documentId"))
	if err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, ErrNotFound)
		return
	}

	c, err := h.sys.Classify(r.Context(), documentID)
	if err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(err), err)
		return
	}

	handlers.RespondJSON(w, http.StatusCreated, c)
}

func (h *Handler) Validate(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, ErrNotFound)
		return
	}

	var cmd ValidateCommand
	if err := json.NewDecoder(r.Body).Decode(&cmd); err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, err)
		return
	}

	c, err := h.sys.Validate(r.Context(), id, cmd)
	if err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(err), err)
		return
	}

	handlers.RespondJSON(w, http.StatusOK, c)
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

	c, err := h.sys.Update(r.Context(), id, cmd)
	if err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(err), err)
		return
	}

	handlers.RespondJSON(w, http.StatusOK, c)
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
```

### Step 4: Wire classifications into API domain

**`internal/api/domain.go`** — add the Classifications field. No workflow import needed — classifications owns the runtime construction:

```go
package api

import (
	"github.com/JaimeStill/herald/internal/classifications"
	"github.com/JaimeStill/herald/internal/documents"
	"github.com/JaimeStill/herald/internal/prompts"
)

type Domain struct {
	Documents       documents.System
	Prompts         prompts.System
	Classifications classifications.System
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

	classificationsSystem := classifications.New(
		runtime.Database.Connection(),
		runtime.Agent,
		runtime.Logger,
		runtime.Pagination,
		runtime.Storage,
		docsSystem,
		promptsSystem,
	)

	return &Domain{
		Documents:       docsSystem,
		Prompts:         promptsSystem,
		Classifications: classificationsSystem,
	}
}
```

### Step 5: Register classifications routes

**`internal/api/routes.go`** — add classifications route group:

```go
func registerRoutes(
	mux *http.ServeMux,
	domain *Domain,
	cfg *config.Config,
	runtime *Runtime,
) {

	documentsRoutes := domain.
		Documents.
		Handler(cfg.API.MaxUploadSizeBytes()).
		Routes()

	promptsRoutes := domain.
		Prompts.
		Handler().
		Routes()

	classificationsRoutes := domain.
		Classifications.
		Handler().
		Routes()

	storageRoutes := newStorageHandler(
		runtime.Storage,
		runtime.Logger,
		cfg.Storage.MaxListSize,
	).routes()

	routes.Register(
		mux,
		documentsRoutes,
		promptsRoutes,
		classificationsRoutes,
		storageRoutes,
	)
}
```

## Validation Criteria

- [ ] `mise run vet` passes
- [ ] `mise run test` passes (existing tests unaffected)
- [ ] Server starts without errors — `domain.Classifications` is non-nil
- [ ] All 8 HTTP endpoints respond correctly
