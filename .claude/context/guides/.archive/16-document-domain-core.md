# 16 - Document Domain Core

## Problem Context

Herald needs a document domain foundation layer — types, data access, and business logic — before HTTP handlers can be built (issue #17). This establishes the complete `internal/documents/` package following the System/Repository/Mapping/Errors pattern adapted from agent-lab.

## Architecture Approach

The repository holds `*sql.DB`, `storage.System`, `*slog.Logger`, and `pagination.Config`. All queries go through `pkg/query.Builder` with a `ProjectionMap`. Blob+DB atomicity uses upload-first with compensating delete on DB failure. No HTTP handler concerns — that's issue #17. No `Search` method on the repository — the GET list vs POST search distinction is a handler concern; both will call `List`.

## Implementation

### Step 1: Add uuid dependency

```bash
go get github.com/google/uuid
```

### Step 2: Create `internal/documents/document.go`

Replace the existing `doc.go` stub with this file.

```go
package documents

import (
	"time"

	"github.com/google/uuid"
)

type Document struct {
	ID               uuid.UUID `json:"id"`
	ExternalID       int       `json:"external_id"`
	ExternalPlatform string    `json:"external_platform"`
	Filename         string    `json:"filename"`
	ContentType      string    `json:"content_type"`
	SizeBytes        int64     `json:"size_bytes"`
	PageCount        *int      `json:"page_count"`
	StorageKey       string    `json:"storage_key"`
	Status           string    `json:"status"`
	UploadedAt       time.Time `json:"uploaded_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type CreateCommand struct {
	Data             []byte
	Filename         string
	ContentType      string
	ExternalID       int
	ExternalPlatform string
	PageCount        *int
}

type BatchResult struct {
	Document *Document `json:"document,omitempty"`
	Filename string    `json:"filename"`
	Error    string    `json:"error,omitempty"`
}
```

### Step 3: Create `internal/documents/errors.go`

```go
package documents

import (
	"errors"
	"net/http"
)

var (
	ErrNotFound    = errors.New("document not found")
	ErrDuplicate   = errors.New("document already exists")
	ErrFileTooLarge = errors.New("file exceeds maximum upload size")
	ErrInvalidFile = errors.New("invalid file")
)

func MapHTTPStatus(err error) int {
	if errors.Is(err, ErrNotFound) {
		return http.StatusNotFound
	}
	if errors.Is(err, ErrDuplicate) {
		return http.StatusConflict
	}
	if errors.Is(err, ErrFileTooLarge) {
		return http.StatusRequestEntityTooLarge
	}
	if errors.Is(err, ErrInvalidFile) {
		return http.StatusBadRequest
	}
	return http.StatusInternalServerError
}
```

### Step 4: Create `internal/documents/mapping.go`

```go
package documents

import (
	"net/url"
	"strconv"

	"github.com/JaimeStill/herald/pkg/query"
	"github.com/JaimeStill/herald/pkg/repository"
)

var projection = query.
	NewProjectionMap("public", "documents", "d").
	Project("id", "ID").
	Project("external_id", "ExternalID").
	Project("external_platform", "ExternalPlatform").
	Project("filename", "Filename").
	Project("content_type", "ContentType").
	Project("size_bytes", "SizeBytes").
	Project("page_count", "PageCount").
	Project("storage_key", "StorageKey").
	Project("status", "Status").
	Project("uploaded_at", "UploadedAt").
	Project("updated_at", "UpdatedAt")

var defaultSort = query.SortField{Field: "UploadedAt", Descending: true}

func scanDocument(s repository.Scanner) (Document, error) {
	var d Document
	err := s.Scan(
		&d.ID,
		&d.ExternalID,
		&d.ExternalPlatform,
		&d.Filename,
		&d.ContentType,
		&d.SizeBytes,
		&d.PageCount,
		&d.StorageKey,
		&d.Status,
		&d.UploadedAt,
		&d.UpdatedAt,
	)
	return d, err
}

type Filters struct {
	Status           *string `json:"status,omitempty"`
	Filename         *string `json:"filename,omitempty"`
	ExternalID       *int    `json:"external_id,omitempty"`
	ExternalPlatform *string `json:"external_platform,omitempty"`
	ContentType      *string `json:"content_type,omitempty"`
	StorageKey       *string `json:"storage_key,omitempty"`
}

func FiltersFromQuery(values url.Values) Filters {
	var f Filters

	if s := values.Get("status"); s != "" {
		f.Status = &s
	}

	if fn := values.Get("filename"); fn != "" {
		f.Filename = &fn
	}

	if eid := values.Get("external_id"); eid != "" {
		if v, err := strconv.Atoi(eid); err == nil {
			f.ExternalID = &v
		}
	}

	if ep := values.Get("external_platform"); ep != "" {
		f.ExternalPlatform = &ep
	}

	if ct := values.Get("content_type"); ct != "" {
		f.ContentType = &ct
	}

	if sk := values.Get("storage_key"); sk != "" {
		f.StorageKey = &sk
	}

	return f
}

func (f Filters) Apply(b *query.Builder) *query.Builder {
	return b.
		WhereEquals("Status", f.Status).
		WhereContains("Filename", f.Filename).
		WhereEquals("ExternalID", f.ExternalID).
		WhereEquals("ExternalPlatform", f.ExternalPlatform).
		WhereEquals("ContentType", f.ContentType).
		WhereContains("StorageKey", f.StorageKey)
}
```

### Step 5: Create `internal/documents/system.go`

```go
package documents

import (
	"context"

	"github.com/google/uuid"

	"github.com/JaimeStill/herald/pkg/pagination"
)

type System interface {
	List(ctx context.Context, page pagination.PageRequest, filters Filters) (*pagination.PageResult[Document], error)
	Find(ctx context.Context, id uuid.UUID) (*Document, error)
	Create(ctx context.Context, cmd CreateCommand) (*Document, error)
	CreateBatch(ctx context.Context, cmds []CreateCommand) []BatchResult
	Delete(ctx context.Context, id uuid.UUID) error
}
```

### Step 6: Create `internal/documents/repository.go`

```go
package documents

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/url"
	"path/filepath"

	"github.com/google/uuid"

	"github.com/JaimeStill/herald/pkg/pagination"
	"github.com/JaimeStill/herald/pkg/query"
	"github.com/JaimeStill/herald/pkg/repository"
	"github.com/JaimeStill/herald/pkg/storage"
)

type repo struct {
	db         *sql.DB
	storage    storage.System
	logger     *slog.Logger
	pagination pagination.Config
}

func New(db *sql.DB, store storage.System, logger *slog.Logger, pagination pagination.Config) System {
	return &repo{
		db:         db,
		storage:    store,
		logger:     logger.With("system", "documents"),
		pagination: pagination,
	}
}

func (r *repo) List(ctx context.Context, page pagination.PageRequest, filters Filters) (*pagination.PageResult[Document], error) {
	page.Normalize(r.pagination)

	qb := query.
		NewBuilder(projection, defaultSort).
		WhereSearch(page.Search, "Filename", "ExternalPlatform")

	filters.Apply(qb)

	if len(page.Sort) > 0 {
		qb.OrderByFields(page.Sort)
	}

	countSQL, countArgs := qb.BuildCount()
	var total int
	if err := r.db.QueryRowContext(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count documents: %w", err)
	}

	pageSQL, pageArgs := qb.BuildPage(page.Page, page.PageSize)
	docs, err := repository.QueryMany(ctx, r.db, pageSQL, pageArgs, scanDocument)
	if err != nil {
		return nil, fmt.Errorf("query documents: %w", err)
	}

	result := pagination.NewPageResult(docs, total, page.Page, page.PageSize)
	return &result, nil
}

func (r *repo) Find(ctx context.Context, id uuid.UUID) (*Document, error) {
	q, args := query.NewBuilder(projection).BuildSingle("ID", id)

	d, err := repository.QueryOne(ctx, r.db, q, args, scanDocument)
	if err != nil {
		return nil, repository.MapError(err, ErrNotFound, ErrDuplicate)
	}
	return &d, nil
}

func (r *repo) Create(ctx context.Context, cmd CreateCommand) (*Document, error) {
	id := uuid.New()
	key := buildStorageKey(id, sanitizeFilename(cmd.Filename))

	if err := r.storage.Upload(ctx, key, bytes.NewReader(cmd.Data), cmd.ContentType); err != nil {
		return nil, fmt.Errorf("upload document blob: %w", err)
	}

	q := `
		INSERT INTO documents (id, external_id, external_platform, filename, content_type, size_bytes, page_count, storage_key)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, external_id, external_platform, filename, content_type, size_bytes, page_count, storage_key, status, uploaded_at, updated_at`

	insertArgs := []any{
		id,
		cmd.ExternalID,
		cmd.ExternalPlatform,
		cmd.Filename,
		cmd.ContentType,
		int64(len(cmd.Data)),
		cmd.PageCount,
		key,
	}

	d, err := repository.WithTx(ctx, r.db, func(tx *sql.Tx) (Document, error) {
		return repository.QueryOne(ctx, tx, q, insertArgs, scanDocument)
	})

	if err != nil {
		// compensating blob delete — log but tolerate failure
		if delErr := r.storage.Delete(ctx, key); delErr != nil {
			r.logger.Warn("compensating blob delete failed", "key", key, "error", delErr)
		}
		return nil, repository.MapError(err, ErrNotFound, ErrDuplicate)
	}

	r.logger.Info("document created", "id", d.ID, "filename", d.Filename)
	return &d, nil
}

func (r *repo) CreateBatch(ctx context.Context, cmds []CreateCommand) []BatchResult {
	results := make([]BatchResult, len(cmds))

	for i, cmd := range cmds {
		doc, err := r.Create(ctx, cmd)
		if err != nil {
			results[i] = BatchResult{
				Filename: cmd.Filename,
				Error:    err.Error(),
			}
		} else {
			results[i] = BatchResult{
				Document: doc,
				Filename: cmd.Filename,
			}
		}
	}

	return results
}

func (r *repo) Delete(ctx context.Context, id uuid.UUID) error {
	doc, err := r.Find(ctx, id)
	if err != nil {
		return err
	}

	_, err = repository.WithTx(ctx, r.db, func(tx *sql.Tx) (struct{}, error) {
		if err := repository.ExecExpectOne(ctx, tx, "DELETE FROM documents WHERE id = $1", id); err != nil {
			return struct{}{}, err
		}
		return struct{}{}, nil
	})

	if err != nil {
		return repository.MapError(err, ErrNotFound, ErrDuplicate)
	}

	if delErr := r.storage.Delete(ctx, doc.StorageKey); delErr != nil {
		r.logger.Warn("blob delete failed after DB delete", "key", doc.StorageKey, "error", delErr)
	}

	r.logger.Info("document deleted", "id", id)
	return nil
}

func buildStorageKey(id uuid.UUID, filename string) string {
	return fmt.Sprintf("documents/%s/%s", id, filename)
}

func sanitizeFilename(name string) string {
	name = filepath.Base(name)
	if name == "." || name == "" {
		name = "document"
	}
	return url.PathEscape(name)
}
```

### Step 7: Delete `internal/documents/doc.go`

Remove the stub file now that the package has real source files.

```bash
rm internal/documents/doc.go
```

### Step 8: Wire domain into `internal/api/domain.go`

Add the `Documents` field and wire the constructor:

```go
package api

import "github.com/JaimeStill/herald/internal/documents"

type Domain struct {
	Documents documents.System
}

func NewDomain(runtime *Runtime) *Domain {
	return &Domain{
		Documents: documents.New(
			runtime.Database.Connection(),
			runtime.Storage,
			runtime.Logger,
			runtime.Pagination,
		),
	}
}
```

## Validation Criteria

- [ ] `go get github.com/google/uuid` succeeds
- [ ] All domain types match the documents DB schema (migration 000001)
- [ ] ProjectionMap columns match migration 000001 column order
- [ ] `go vet ./...` passes
- [ ] `go build ./...` compiles cleanly
- [ ] `internal/documents/doc.go` is removed
