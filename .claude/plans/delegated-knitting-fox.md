# Objective Planning: #3 Document Domain

## Context

Objective #3 is the final Phase 1 objective. Objectives #1 (scaffolding) and #2 (migration CLI + schema) are complete. The full infrastructure foundation is in place — database, storage, lifecycle, configuration, routing, middleware, query builder, repository helpers, pagination, OpenAPI types. This objective builds the first domain system on top of that foundation.

## Step 0: Transition Closeout

Objective #2 is 100% complete (sub-issue #13 closed) but the objective issue itself is still OPEN.

1. Close issue #2 on GitHub
2. Update `_project/phase.md` — mark #2 status as `Complete`
3. Delete `_project/objective.md` (will be recreated for #3)

## Architecture Decisions

### Endpoints

| Method | Path | Purpose |
|--------|------|---------|
| `GET /documents` | Paginated list with query param filters (status, page, sort) |
| `GET /documents/{id}` | Find by ID |
| `POST /documents` | Single upload (multipart/form-data) |
| `POST /documents/batch` | Batch upload (multipart, multiple files) |
| `POST /documents/search` | Search with JSON body for richer multi-field criteria |
| `DELETE /documents/{id}` | Delete document (DB + blob) |

### GET list vs POST search

- `GET /documents` — simple filtering via URL query params (page, page_size, sort, status)
- `POST /documents/search` — JSON body with richer filter combinations (multiple fields, date ranges, text search)
- Both return `PageResult[Document]`

### Batch upload semantics

Partial success — each file processed independently. Response includes per-file result (success with document JSON, or error with message). This is practical for the 1M-document bulk ingestion use case.

### Storage key construction

`documents/{id}/{filename}.pdf` — UUID generated before insert, used in both blob key and DB record.

### Blob + DB atomicity (single upload)

1. Upload blob to storage first
2. Insert DB record within `repository.WithTx`
3. On DB failure, compensating delete of the blob (log but tolerate `ErrNotFound`)

### PDF page count

`pdfcpu` library (`github.com/pdfcpu/pdfcpu`) — extract page count from uploaded bytes. Non-fatal on failure (log warning, set `page_count` to nil).

### Max upload size

Add `MaxUploadSize` to API config (`internal/config/api.go`) with env var `HERALD_API_MAX_UPLOAD_SIZE`. Default: `"50MB"`. Parsed to bytes at finalize time.

### Domain patterns (adapted from agent-lab)

- **System interface** defined in `system.go` — consumed by handler
- **Repository** implements System — holds `*sql.DB`, `storage.System`, `*slog.Logger`
- **Handler** holds System + logger + config — HTTP concerns only
- **Mapping** (`mapping.go`) — ProjectionMap, scanDocument, Filters, FiltersFromQuery
- **Errors** (`errors.go`) — typed domain errors, MapHTTPStatus helper
- **OpenAPI** (`openapi.go`) — package-level Spec variable with Operations and Schemas

## Sub-Issue Decomposition

### Sub-Issue 1: Document domain core — types, mapping, repository, and system

**Labels:** `feature`, `task`
**Milestone:** v0.1.0 - Service Foundation

**Scope:**
- `internal/documents/document.go` — Document struct (matching DB schema), CreateCommand, BatchResult types
- `internal/documents/errors.go` — ErrNotFound, ErrDuplicate, ErrFileTooLarge, ErrInvalidFile, MapHTTPStatus
- `internal/documents/mapping.go` — ProjectionMap (all documents columns), scanDocument ScanFunc, defaultSort, Filters struct, FiltersFromQuery, Apply method on Builder
- `internal/documents/repository.go` — repo struct, New(), List (paginated+filtered), Find (by ID), Create (blob upload + DB insert with atomicity), CreateBatch (parallel per-file processing with partial success), Delete (DB + blob), Search (from Filters), buildStorageKey, sanitizeFilename
- `internal/documents/system.go` — System interface (List, Find, Create, CreateBatch, Delete, Search, Handler), repo satisfies it

**Acceptance criteria:**
- All domain types match the documents DB schema
- Repository CRUD operations work against `*sql.DB` and `storage.System`
- Create handles blob+DB atomicity with compensating delete
- CreateBatch processes files independently with per-file results
- ProjectionMap columns match migration 000001
- Filters support status (equals), filename (contains), search (multi-field ILIKE)

### Sub-Issue 2: Document HTTP handlers, OpenAPI specs, and API wiring

**Labels:** `feature`, `task`
**Milestone:** v0.1.0 - Service Foundation
**Dependencies:** Sub-issue 1

**Scope:**
- `internal/documents/handler.go` — Handler struct, NewHandler, all 6 HTTP endpoint handlers:
  - `List` — parse PageRequestFromQuery, call sys.List, respond with PageResult
  - `Find` — parse `{id}` path param, call sys.Find, respond with Document
  - `Upload` — parse multipart form, extract file, detect content type, extract PDF page count via pdfcpu, build CreateCommand, call sys.Create, respond 201
  - `BatchUpload` — parse multipart form with multiple files, process each independently, call sys.CreateBatch, respond 201 with BatchResult array
  - `Search` — decode JSON body into search request, call sys.Search, respond with PageResult
  - `Delete` — parse `{id}` path param, call sys.Delete, respond 204
- `internal/documents/openapi.go` — Spec variable with all Operations, Schemas() method for component registration
- Handler `Routes()` method returning `routes.Group`
- `internal/config/api.go` — add MaxUploadSize field with env var and parse helper
- `internal/api/domain.go` — add Documents field, wire `documents.New()` in NewDomain
- `internal/api/routes.go` — register document routes via `routes.Register`, add schemas via `spec.Components.AddSchemas`
- `go.mod` — add `github.com/pdfcpu/pdfcpu`

**Acceptance criteria:**
- `go vet ./...` passes
- Upload single PDF → 201 with document JSON (ID, storage key, page count, status "pending")
- Upload batch → 201 with array of per-file results
- GET /documents → paginated results with filters
- POST /documents/search → filtered results from JSON body
- GET /documents/{id} → single document
- DELETE /documents/{id} → 204, blob removed from storage
- OpenAPI spec at /api/openapi.json includes all document endpoints and schemas

## Step 5: `_project/objective.md`

Create with:
- Objective title (#3) and phase reference
- Scope and acceptance criteria (from issue body)
- Sub-issues table with status and dependencies
- Architecture decisions (endpoints, atomicity, batch semantics, search vs list)

## Step 6: Project Board Updates

- Add each sub-issue to project #7 (Herald)
- Assign Phase 1 to each sub-issue
- Verify milestone `v0.1.0 - Service Foundation` is set

## Execution Order

1. Transition closeout (close #2, update phase.md, delete objective.md)
2. Create sub-issue 1 on GitHub, link to #3
3. Create sub-issue 2 on GitHub, link to #3
4. Add both to project board with Phase 1
5. Create `_project/objective.md`
