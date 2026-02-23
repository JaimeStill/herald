# Plan: Issue #16 — Document Domain Core

## Context

Issue #16 establishes the document domain's foundation layer in `internal/documents/` — all types, data access, and business logic with no HTTP concerns. This is sub-issue 1 of Objective #3 (Document Domain). Sub-issue #17 builds the HTTP handler layer on top of this work.

The domain follows the System/Repository/Mapping/Errors pattern from agent-lab, adapted for Herald. The repository holds `*sql.DB` (from `database.System.Connection()`), `storage.System`, and `*slog.Logger`, using the existing `pkg/query`, `pkg/repository`, and `pkg/pagination` infrastructure.

## New Files

All files created in `internal/documents/` (replacing the existing `doc.go` stub).

### 1. `document.go` — Domain types

- **`Document`** struct matching the DB schema from migration 000001:
  - `ID uuid.UUID`, `ExternalID int`, `ExternalPlatform string`, `Filename string`, `ContentType string`, `SizeBytes int64`, `PageCount *int`, `StorageKey string`, `Status string`, `UploadedAt time.Time`, `UpdatedAt time.Time`
  - JSON tags on all fields for API serialization

- **`CreateCommand`** struct — input for creating a document:
  - `Data []byte`, `Filename string`, `ContentType string`, `ExternalID int`, `ExternalPlatform string`, `PageCount *int`
  - `Data` is the raw file bytes (used for blob upload and size calculation)
  - `PageCount` is optional — extracted by the handler (issue #17) via pdfcpu; nil on failure

- **`BatchResult`** struct — per-file result from batch create:
  - `Document *Document`, `Filename string`, `Error string`

### 2. `errors.go` — Domain errors and HTTP status mapping

- Sentinel errors: `ErrNotFound`, `ErrDuplicate`, `ErrFileTooLarge`, `ErrInvalidFile`
- `MapHTTPStatus(error) int` — maps domain errors to HTTP status codes:
  - `ErrNotFound` → 404, `ErrDuplicate` → 409, `ErrFileTooLarge` → 413, `ErrInvalidFile` → 400
  - Default → 500

### 3. `mapping.go` — Query projections, scanning, and filters

- **`projection`** — `query.NewProjectionMap("public", "documents", "d")` with all 11 columns mapped in migration order:
  - `id→ID`, `external_id→ExternalID`, `external_platform→ExternalPlatform`, `filename→Filename`, `content_type→ContentType`, `size_bytes→SizeBytes`, `page_count→PageCount`, `storage_key→StorageKey`, `status→Status`, `uploaded_at→UploadedAt`, `updated_at→UpdatedAt`

- **`defaultSort`** — `query.SortField{Field: "UploadedAt", Descending: true}` (newest first)

- **`scanDocument`** — `repository.ScanFunc[Document]` scanning all 11 columns in projection order

- **`Filters`** struct:
  - `Status *string` — exact match (`WhereEquals`)
  - `Filename *string` — contains match (`WhereContains`)
  - `ExternalPlatform *string` — exact match (`WhereEquals`)

- **`FiltersFromQuery(url.Values) Filters`** — parses query params to Filters
- **`(Filters).Apply(*query.Builder) *query.Builder`** — adds filter conditions

### 4. `repository.go` — Data access implementation

- **`repo`** struct: `db *sql.DB`, `storage storage.System`, `logger *slog.Logger`, `pagination pagination.Config`
- **`New(db, storage, logger, pagination) System`** constructor — returns the System interface

**Methods:**

- **`List(ctx, PageRequest, Filters) (*PageResult[Document], error)`**
  Normalize page → NewBuilder with projection/defaultSort → WhereSearch on Filename, ExternalPlatform → filters.Apply → BuildCount + BuildPage → return PageResult

- **`Find(ctx, uuid.UUID) (*Document, error)`**
  BuildSingle by ID → QueryOne → MapError

- **`Create(ctx, CreateCommand) (*Document, error)`**
  1. Generate `id := uuid.New()`
  2. Sanitize filename, build storage key: `documents/{id}/{sanitized}`
  3. Upload blob: `r.storage.Upload(ctx, key, bytes.NewReader(cmd.Data), cmd.ContentType)`
  4. INSERT in WithTx with RETURNING (all columns)
  5. On DB failure: compensating `r.storage.Delete` (log warning on delete failure)
  6. Return the created Document

- **`CreateBatch(ctx, []CreateCommand) []BatchResult`**
  Process each command independently via `r.Create()`. Collect BatchResult per file — success with Document, or error with message. Partial success semantics.

- **`Delete(ctx, uuid.UUID) error`**
  1. Find document (need storage_key)
  2. Delete DB record via ExecExpectOne in WithTx
  3. Delete blob (log warning if blob delete fails — orphan is acceptable)
  4. MapError on DB errors

No `Search` method — the GET list vs POST search distinction is an HTTP handler concern (issue #17). Both call the same `List` method with different filter sources.

**Helpers (unexported):**

- **`buildStorageKey(id uuid.UUID, filename string) string`** — `fmt.Sprintf("documents/%s/%s", id, filename)`
- **`sanitizeFilename(string) string`** — strips path separators, ".." sequences, leading dots; preserves extension

### 5. `system.go` — System interface

Defines the public contract:
- `List(ctx, PageRequest, Filters) (*PageResult[Document], error)`
- `Find(ctx, uuid.UUID) (*Document, error)`
- `Create(ctx, CreateCommand) (*Document, error)`
- `CreateBatch(ctx, []CreateCommand) []BatchResult`
- `Delete(ctx, uuid.UUID) error`

No `Handler()` or `Search()` — Handler is added in #17, and the GET/POST search distinction is a handler concern (both call `List`).

## Modified Files

### `internal/api/domain.go`

Add `Documents documents.System` field to Domain struct. Wire in NewDomain:
```go
Documents: documents.New(
    runtime.Database.Connection(),
    runtime.Storage,
    runtime.Logger,
    runtime.Pagination,
),
```

### `go.mod` / `go.sum`

Add `github.com/google/uuid` dependency (needed for `uuid.UUID` type in Document struct and `uuid.New()` in Create).

## Dependencies Reused

| Package | Used For |
|---------|----------|
| `pkg/query` | ProjectionMap, Builder, SortField, WhereEquals/WhereContains/WhereSearch |
| `pkg/repository` | WithTx, QueryOne, QueryMany, ExecExpectOne, MapError, Scanner, ScanFunc |
| `pkg/pagination` | PageRequest, PageResult, NewPageResult, Config |
| `pkg/storage` | System interface (Upload, Delete) |
| `pkg/database` | System.Connection() for *sql.DB |

## Validation

- `go vet ./...` passes
- `go build ./...` compiles cleanly
- All domain types match documents DB schema (migration 000001)
- ProjectionMap columns match migration column order
- Repository methods compile against `*sql.DB` and `storage.System`
- Filters support status (equals), filename (contains), external_platform (equals)
