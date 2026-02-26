# 47 - Classifications Domain: Types, System, and Repository

## Problem Context

Objective #27 (Classifications Domain) requires the complete data layer for classification persistence. This includes all types, errors, query mapping, the System interface, and the repository implementation. The repository holds a `*workflow.Runtime` dependency to support `Classify`, which calls `workflow.Execute()` internally.

## Architecture Approach

Follow the established domain patterns from `internal/documents/` and `internal/prompts/`. The classifications domain mirrors these patterns with the addition of workflow integration (`Classify`), transactional document status transitions (`Validate`, `Update`), and JSONB column handling (`markings_found`).

Key decisions from planning:
- **Move `workflow/` to `internal/workflow/`** — it already imports internal packages, so formalize it
- **Confidence as plain string** — decouples persistence from workflow types; DB CHECK constraint validates
- **UpdateCommand includes UpdatedBy** — both Validate and Update finalize a classification, so both track who did it via `validated_by`/`validated_at`
- **Classify sets status unconditionally** — `UPDATE documents SET status = 'review' WHERE id = $1` supports initial classification and re-classification from any state
- **Validate/Update guard on review status** — `WHERE status = 'review'` on document update, maps 0 rows to ErrInvalidStatus

## Implementation

### Step 0: Move `workflow/` to `internal/workflow/`

Move the workflow package and update all imports.

```bash
mv workflow/ internal/workflow/
```

Update imports in two test files:

**`tests/workflow/types_test.go`** — change import:
```go
// old
"github.com/JaimeStill/herald/workflow"

// new
"github.com/JaimeStill/herald/internal/workflow"
```

**`tests/workflow/prompts_test.go`** — change import:
```go
// old
"github.com/JaimeStill/herald/workflow"

// new
"github.com/JaimeStill/herald/internal/workflow"
```

Verify after move:
```bash
go build ./internal/workflow/...
mise run test
```

### Step 1: Delete `doc.go`, create `classification.go`

Delete the existing stub:
```bash
rm internal/classifications/doc.go
```

**`internal/classifications/classification.go`** — new file:

```go
package classifications

import (
	"time"

	"github.com/google/uuid"
)

type Classification struct {
	ID             uuid.UUID  `json:"id"`
	DocumentID     uuid.UUID  `json:"document_id"`
	Classification string     `json:"classification"`
	Confidence     string     `json:"confidence"`
	MarkingsFound  []string   `json:"markings_found"`
	Rationale      string     `json:"rationale"`
	ClassifiedAt   time.Time  `json:"classified_at"`
	ModelName      string     `json:"model_name"`
	ProviderName   string     `json:"provider_name"`
	ValidatedBy    *string    `json:"validated_by"`
	ValidatedAt    *time.Time `json:"validated_at"`
}

type ValidateCommand struct {
	ValidatedBy string `json:"validated_by"`
}

type UpdateCommand struct {
	Classification string `json:"classification"`
	Rationale      string `json:"rationale"`
	UpdatedBy      string `json:"updated_by"`
}
```

### Step 2: Create `errors.go`

**`internal/classifications/errors.go`** — new file:

```go
package classifications

import (
	"errors"
	"net/http"
)

var (
	ErrNotFound      = errors.New("classification not found")
	ErrDuplicate     = errors.New("classification already exists")
	ErrInvalidStatus = errors.New("document is not in review status")
)

func MapHTTPStatus(err error) int {
	if errors.Is(err, ErrNotFound) {
		return http.StatusNotFound
	}
	if errors.Is(err, ErrDuplicate) {
		return http.StatusConflict
	}
	if errors.Is(err, ErrInvalidStatus) {
		return http.StatusConflict
	}
	return http.StatusInternalServerError
}
```

### Step 3: Create `system.go`

**`internal/classifications/system.go`** — new file:

```go
package classifications

import (
	"context"

	"github.com/google/uuid"

	"github.com/JaimeStill/herald/pkg/pagination"
)

type System interface {
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

No `Handler` type or method — both added together in #48.

### Step 4: Create `mapping.go`

**`internal/classifications/mapping.go`** — new file:

```go
package classifications

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/google/uuid"

	"github.com/JaimeStill/herald/pkg/query"
	"github.com/JaimeStill/herald/pkg/repository"
)

var projection = query.
	NewProjectionMap("public", "classifications", "c").
	Project("id", "ID").
	Project("document_id", "DocumentID").
	Project("classification", "Classification").
	Project("confidence", "Confidence").
	Project("markings_found", "MarkingsFound").
	Project("rationale", "Rationale").
	Project("classified_at", "ClassifiedAt").
	Project("model_name", "ModelName").
	Project("provider_name", "ProviderName").
	Project("validated_by", "ValidatedBy").
	Project("validated_at", "ValidatedAt")

var defaultSort = query.SortField{
	Field:      "ClassifiedAt",
	Descending: true,
}

type Filters struct {
	Classification *string    `json:"classification,omitempty"`
	Confidence     *string    `json:"confidence,omitempty"`
	DocumentID     *uuid.UUID `json:"document_id,omitempty"`
	ValidatedBy    *string    `json:"validated_by,omitempty"`
}

func (f Filters) Apply(b *query.Builder) *query.Builder {
	return b.
		WhereEquals("Classification", f.Classification).
		WhereEquals("Confidence", f.Confidence).
		WhereEquals("DocumentID", f.DocumentID).
		WhereEquals("ValidatedBy", f.ValidatedBy)
}

func FiltersFromQuery(values url.Values) Filters {
	var f Filters

	if c := values.Get("classification"); c != "" {
		f.Classification = &c
	}

	if c := values.Get("confidence"); c != "" {
		f.Confidence = &c
	}

	if d := values.Get("document_id"); d != "" {
		if id, err := uuid.Parse(d); err == nil {
			f.DocumentID = &id
		}
	}

	if v := values.Get("validated_by"); v != "" {
		f.ValidatedBy = &v
	}

	return f
}

func scanClassification(s repository.Scanner) (Classification, error) {
	var c Classification
	var markingsRaw []byte

	err := s.Scan(
		&c.ID,
		&c.DocumentID,
		&c.Classification,
		&c.Confidence,
		&markingsRaw,
		&c.Rationale,
		&c.ClassifiedAt,
		&c.ModelName,
		&c.ProviderName,
		&c.ValidatedBy,
		&c.ValidatedAt,
	)

	if err != nil {
		return c, err
	}

	if len(markingsRaw) > 0 {
		if err := json.Unmarshal(markingsRaw, &c.MarkingsFound); err != nil {
			return c, fmt.Errorf("unmarshal markings_found: %w", err)
		}
	}

	if c.MarkingsFound == nil {
		c.MarkingsFound = []string{}
	}

	return c, nil
}
```

### Step 5: Create `repository.go`

**`internal/classifications/repository.go`** — new file:

```go
package classifications

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"slices"

	"github.com/google/uuid"

	"github.com/JaimeStill/herald/internal/workflow"
	"github.com/JaimeStill/herald/pkg/pagination"
	"github.com/JaimeStill/herald/pkg/query"
	"github.com/JaimeStill/herald/pkg/repository"
)

type repo struct {
	db         *sql.DB
	rt         *workflow.Runtime
	logger     *slog.Logger
	pagination pagination.Config
}

func New(
	db *sql.DB,
	rt *workflow.Runtime,
	logger *slog.Logger,
	pagination pagination.Config,
) System {
	return &repo{
		db:         db,
		rt:         rt,
		logger:     logger.With("system", "classifications"),
		pagination: pagination,
	}
}

func (r *repo) List(
	ctx context.Context,
	page pagination.PageRequest,
	filters Filters,
) (*pagination.PageResult[Classification], error) {
	page.Normalize(r.pagination)

	qb := query.
		NewBuilder(projection, defaultSort).
		WhereSearch(page.Search, "Classification", "Rationale")

	filters.Apply(qb)

	if len(page.Sort) > 0 {
		qb.OrderByFields(page.Sort)
	}

	countSQL, countArgs := qb.BuildCount()
	var total int
	if err := r.db.QueryRowContext(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count classifications: %w", err)
	}

	pageSQL, pageArgs := qb.BuildPage(page.Page, page.PageSize)
	items, err := repository.QueryMany(ctx, r.db, pageSQL, pageArgs, scanClassification)
	if err != nil {
		return nil, fmt.Errorf("query classifications: %w", err)
	}

	result := pagination.NewPageResult(items, total, page.Page, page.PageSize)
	return &result, nil
}

func (r *repo) Find(ctx context.Context, id uuid.UUID) (*Classification, error) {
	q, args := query.NewBuilder(projection).BuildSingle("ID", id)

	c, err := repository.QueryOne(ctx, r.db, q, args, scanClassification)
	if err != nil {
		return nil, repository.MapError(err, ErrNotFound, ErrDuplicate)
	}
	return &c, nil
}

func (r *repo) FindByDocument(ctx context.Context, documentID uuid.UUID) (*Classification, error) {
	q, args := query.NewBuilder(projection).BuildSingle("DocumentID", documentID)

	c, err := repository.QueryOne(ctx, r.db, q, args, scanClassification)
	if err != nil {
		return nil, repository.MapError(err, ErrNotFound, ErrDuplicate)
	}
	return &c, nil
}

func (r *repo) Classify(ctx context.Context, documentID uuid.UUID) (*Classification, error) {
	result, err := workflow.Execute(ctx, r.rt, documentID)
	if err != nil {
		return nil, fmt.Errorf("classify document %s: %w", documentID, err)
	}

	markings := collectMarkings(result.State.Pages)
	markingsJSON, err := json.Marshal(markings)
	if err != nil {
		return nil, fmt.Errorf("marshal markings: %w", err)
	}

	upsertQ := `
		INSERT INTO classifications(
			document_id, classification, confidence, markings_found,
			rationale, model_name, provider_name
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (document_id) DO UPDATE SET
			classification = EXCLUDED.classification,
			confidence = EXCLUDED.confidence,
			markings_found = EXCLUDED.markings_found,
			rationale = EXCLUDED.rationale,
			classified_at = NOW(),
			model_name = EXCLUDED.model_name,
			provider_name = EXCLUDED.provider_name,
			validated_by = NULL,
			validated_at = NULL
		RETURNING id, document_id, classification, confidence, markings_found,
				  rationale, classified_at, model_name, provider_name,
				  validated_by, validated_at`

	upsertArgs := []any{
		documentID,
		result.State.Classification,
		string(result.State.Confidence),
		markingsJSON,
		result.State.Rationale,
		r.rt.Agent.Model.Name,
		r.rt.Agent.Provider.Name,
	}

	c, err := repository.WithTx(ctx, r.db, func(tx *sql.Tx) (Classification, error) {
		cl, err := repository.QueryOne(ctx, tx, upsertQ, upsertArgs, scanClassification)
		if err != nil {
			return Classification{}, fmt.Errorf("upsert classification: %w", err)
		}

		if err := repository.ExecExpectOne(
			ctx, tx,
			"UPDATE documents SET status = 'review', updated_at = NOW() WHERE id = $1",
			documentID,
		); err != nil {
			return Classification{}, fmt.Errorf("update document status: %w", err)
		}

		return cl, nil
	})

	if err != nil {
		return nil, repository.MapError(err, ErrNotFound, ErrDuplicate)
	}

	r.logger.Info("document classified",
		"id", c.ID,
		"document_id", documentID,
		"classification", c.Classification,
		"confidence", c.Confidence,
	)
	return &c, nil
}

func (r *repo) Validate(ctx context.Context, id uuid.UUID, cmd ValidateCommand) (*Classification, error) {
	validateQ := `
		UPDATE classifications
		SET validated_by = $1, validated_at = NOW()
		WHERE id = $2
		RETURNING id, document_id, classification, confidence, markings_found,
				  rationale, classified_at, model_name, provider_name,
				  validated_by, validated_at`

	c, err := repository.WithTx(ctx, r.db, func(tx *sql.Tx) (Classification, error) {
		cl, err := repository.QueryOne(ctx, tx, validateQ, []any{cmd.ValidatedBy, id}, scanClassification)
		if err != nil {
			return Classification{}, repository.MapError(err, ErrNotFound, ErrDuplicate)
		}

		if err := repository.ExecExpectOne(
			ctx, tx,
			"UPDATE documents SET status = 'complete', updated_at = NOW() WHERE id = $1 AND status = 'review'",
			cl.DocumentID,
		); err != nil {
			return Classification{}, ErrInvalidStatus
		}

		return cl, nil
	})

	if err != nil {
		return nil, err
	}

	r.logger.Info("classification validated",
		"id", c.ID,
		"validated_by", c.ValidatedBy,
	)
	return &c, nil
}

func (r *repo) Update(ctx context.Context, id uuid.UUID, cmd UpdateCommand) (*Classification, error) {
	updateQ := `
		UPDATE classifications
		SET classification = $1, rationale = $2, validated_by = $3, validated_at = NOW()
		WHERE id = $4
		RETURNING id, document_id, classification, confidence, markings_found,
				  rationale, classified_at, model_name, provider_name,
				  validated_by, validated_at`

	c, err := repository.WithTx(ctx, r.db, func(tx *sql.Tx) (Classification, error) {
		cl, err := repository.QueryOne(ctx, tx, updateQ,
			[]any{cmd.Classification, cmd.Rationale, cmd.UpdatedBy, id},
			scanClassification,
		)
		if err != nil {
			return Classification{}, repository.MapError(err, ErrNotFound, ErrDuplicate)
		}

		if err := repository.ExecExpectOne(
			ctx, tx,
			"UPDATE documents SET status = 'complete', updated_at = NOW() WHERE id = $1 AND status = 'review'",
			cl.DocumentID,
		); err != nil {
			return Classification{}, ErrInvalidStatus
		}

		return cl, nil
	})

	if err != nil {
		return nil, err
	}

	r.logger.Info("classification updated",
		"id", c.ID,
		"updated_by", cmd.UpdatedBy,
	)
	return &c, nil
}

func (r *repo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := repository.WithTx(ctx, r.db, func(tx *sql.Tx) (struct{}, error) {
		if err := repository.ExecExpectOne(
			ctx, tx,
			"DELETE FROM classifications WHERE id = $1",
			id,
		); err != nil {
			return struct{}{}, err
		}
		return struct{}{}, nil
	})

	if err != nil {
		return repository.MapError(err, ErrNotFound, ErrDuplicate)
	}

	r.logger.Info("classification deleted", "id", id)
	return nil
}

func collectMarkings(pages []workflow.ClassificationPage) []string {
	var all []string
	for _, p := range pages {
		all = append(all, p.MarkingsFound...)
	}

	slices.Sort(all)
	all = slices.Compact(all)

	if all == nil {
		all = []string{}
	}

	return all
}
```

## Validation Criteria

- [ ] Workflow move: `go build ./internal/workflow/...` compiles
- [ ] All tests pass after move: `mise run test`
- [ ] All classification files compile: `go build ./internal/classifications/...`
- [ ] `mise run vet` passes
- [ ] Types match DB schema from migration 000002
- [ ] Upsert uses ON CONFLICT with validation field reset
- [ ] Document status transitions correct: Classify→review, Validate/Update→complete
