# Objective: Document Domain

**Parent Issue:** [#3](https://github.com/JaimeStill/herald/issues/3)
**Phase:** Phase 1 — Service Foundation (v0.1.0)
**Repository:** herald

## Scope

Deliver the complete document lifecycle — upload (single and batch), registration, metadata management, blob storage integration, and paginated list/search/filter queries. At completion, Herald is a fully functional document management API.

### Endpoints

| Method | Path | Purpose |
|--------|------|---------|
| `GET /documents` | Paginated list with query param filters (status, page, sort) |
| `GET /documents/{id}` | Find by ID |
| `POST /documents` | Single upload (multipart/form-data) |
| `POST /documents/batch` | Batch upload (multipart, multiple files) |
| `POST /documents/search` | Search with JSON body for richer multi-field criteria |
| `DELETE /documents/{id}` | Delete document (DB + blob) |

## Sub-Issues

| # | Title | Labels | Status | Dependencies |
|---|-------|--------|--------|--------------|
| [#16](https://github.com/JaimeStill/herald/issues/16) | Document domain core — types, mapping, repository, and system | `feature`, `task` | Open | — |
| [#17](https://github.com/JaimeStill/herald/issues/17) | Document HTTP handlers, OpenAPI specs, and API wiring | `feature`, `task` | Open | #16 |

## Architecture Decisions

### Domain Structure

Follows agent-lab's System/Repository/Handler pattern adapted for Herald:

- **System interface** (`system.go`) — consumed by handler, implemented by repository
- **Repository** (`repository.go`) — holds `*sql.DB`, `storage.System`, `*slog.Logger`; all data access
- **Handler** (`handler.go`) — HTTP concerns only, depends on System interface
- **Mapping** (`mapping.go`) — ProjectionMap, scanDocument, Filters, FiltersFromQuery
- **Errors** (`errors.go`) — typed domain errors, MapHTTPStatus helper
- **OpenAPI** (`openapi.go`) — package-level Spec variable with Operations and Schemas

### GET List vs POST Search

- `GET /documents` — simple filtering via URL query params (page, page_size, sort, status)
- `POST /documents/search` — JSON body with richer filter combinations (multiple fields, date ranges, text search)
- Both return `PageResult[Document]`

### Batch Upload Semantics

Partial success — each file processed independently. Response includes per-file result (success with document JSON, or error with message). Practical for the 1M-document bulk ingestion use case.

### Blob + DB Atomicity

Single upload flow:
1. Upload blob to Azure Blob Storage first
2. Insert DB record within `repository.WithTx`
3. On DB failure, compensating delete of the blob (log but tolerate `ErrNotFound`)

### Storage Key Construction

`documents/{id}/{filename}.pdf` — UUID generated before insert, used in both blob key and DB record.

### PDF Page Count

`pdfcpu` library — extract page count from uploaded PDF bytes. Non-fatal on failure (log warning, set `page_count` to nil).

### Max Upload Size

`MaxUploadSize` on API config (`internal/config/api.go`) with env var `HERALD_API_MAX_UPLOAD_SIZE`. Default: `"50MB"`.

## Verification

- `go vet ./...` passes
- Upload single PDF → 201 with document JSON (ID, storage key, page count, status "pending")
- Upload batch → 201 with array of per-file results
- List with pagination and filters → paginated results
- Search with JSON body → filtered results
- Find by ID → single document
- Delete → 204, blob removed from storage
- OpenAPI spec at /api/openapi.json includes all document endpoints and schemas
