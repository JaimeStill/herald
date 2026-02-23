# 16 - Document Domain Core

## Summary

Established the complete document domain foundation layer in `internal/documents/` — types, data access, business logic, and domain errors — with no HTTP concerns. The domain follows the System/Repository/Mapping/Errors pattern adapted from agent-lab. The repository holds `*sql.DB`, `storage.System`, and `*slog.Logger`, using the existing query builder, repository helpers, and pagination infrastructure. The domain is wired into the API module via `internal/api/domain.go`.

## Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| No Search method on repository | Single `List` method serves both GET list and POST search | The GET vs POST distinction is an HTTP handler concern; both use the same Filters struct |
| No Handler on System interface | Deferred to issue #17 | Handler type doesn't exist yet; added when the HTTP layer is built |
| Filename sanitization | `filepath.Base` + `url.PathEscape` | Standard library functions handle path stripping and URL encoding without manual string manipulation |
| Additional filters | Added ExternalID, ContentType, StorageKey beyond original spec | ExternalID for targeted lookups, ContentType for future doc type expansion, StorageKey for debugging |
| PageCount on CreateCommand | Optional `*int` provided by caller | Extraction via pdfcpu is a handler concern (issue #17); repository stores whatever value is given |

## Files Modified

- `internal/documents/document.go` — Document, CreateCommand, BatchResult types (replaced doc.go stub)
- `internal/documents/errors.go` — ErrNotFound, ErrDuplicate, ErrFileTooLarge, ErrInvalidFile, MapHTTPStatus
- `internal/documents/mapping.go` — ProjectionMap, scanDocument, Filters, FiltersFromQuery, Apply
- `internal/documents/system.go` — System interface (List, Find, Create, CreateBatch, Delete)
- `internal/documents/repository.go` — repo struct, New constructor, all System methods, buildStorageKey, sanitizeFilename
- `internal/api/domain.go` — Added Documents field and wired constructor
- `go.mod` / `go.sum` — Added google/uuid as direct dependency
- `tests/documents/documents_test.go` — Tests for MapHTTPStatus, FiltersFromQuery, Filters.Apply

## Patterns Established

- Domain package structure: document.go (types), errors.go, mapping.go, system.go, repository.go
- Repository constructor: `New(db, storage, logger, pagination) System`
- Blob+DB atomicity: upload blob first, INSERT in WithTx, compensating delete on failure
- Filters with `Apply(*query.Builder)` method and `FiltersFromQuery(url.Values)` constructor
- `sanitizeFilename` using `filepath.Base` + `url.PathEscape`

## Validation Results

- `go vet ./...` passes
- `go build ./...` compiles cleanly
- `go mod tidy` clean (promoted google/uuid to direct dependency)
- 18 new tests pass (MapHTTPStatus: 7, FiltersFromQuery: 4, FiltersApply: 7)
- Full test suite: 15 packages pass
- All 11 Document struct fields match migration 000001 column types and order
