# 21 - Blob Storage Query Endpoints

## Summary

Added three read-only HTTP endpoints under `/api/storage` that query Azure Blob Storage directly, bypassing the SQL layer. Extended the `storage.System` interface with `List`, `Find`, and updated `Download` to return a richer `BlobResult` type. Created a storage handler in `internal/api/` wired directly to the storage system without a domain layer.

## Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Type consolidation | Single `BlobMeta` type + `BlobResult` embedding | Eliminates redundancy between list items, properties, and download metadata |
| `Download` return type | `*BlobResult` (embeds `BlobMeta` + `Body`) | Surfaces content type and length from a single Azure call; method was unused so signature change was safe |
| No domain system | Handler talks directly to `storage.System` | Pure pass-through queries; domain system would be unnecessary indirection |
| Handler placement | Unexported `storageHandler` in `internal/api/` | No separate package needed without a domain system |
| Marker-based pagination | `BlobList` with `NextMarker`, not `PageResult` | Azure Blob Storage uses opaque continuation tokens, not offset-based pages |
| Path parameters for keys | `/{key...}` and `/download/{key...}` wildcards | Blob keys contain slashes; wildcard captures full path cleanly |
| `MaxListSize` config | Configurable default with `MaxListCap` constant (5000) | Consistent with pagination config pattern; deployments can tune the default |
| `ParseMaxResults` in `pkg/storage` | Encapsulates parsing, validation, and capping | Keeps handler clean; reusable; upper limit defined once as `MaxListCap` |
| `MapHTTPStatus` in `pkg/storage/errors.go` | Co-located with error definitions | Consistent with `documents.MapHTTPStatus` pattern |
| Inline nil checks | No `deref` helper functions | Explicit value handling at the Azure SDK boundary; avoids hiding behavior |

## Files Modified

- `pkg/storage/storage.go` — `BlobMeta`, `BlobList`, `BlobResult` types; `List`, `Find`, `ParseMaxResults`; updated `Download` signature and implementation
- `pkg/storage/config.go` — `MaxListSize` field, env override, cap at `MaxListCap`
- `pkg/storage/errors.go` — `MapHTTPStatus` function
- `internal/api/storage.go` — new `storageHandler` with `list`, `find`, `download` endpoints
- `internal/api/routes.go` — wired storage handler, added `runtime` parameter
- `internal/api/api.go` — passed `runtime` to `registerRoutes`
- `internal/config/config.go` — `HERALD_STORAGE_MAX_LIST_SIZE` env var mapping
- `_project/api/storage.md` — API Cartographer spec for storage endpoints
- `_project/api/README.md` — added storage group to route table
- `tests/storage/storage_test.go` — `TestMapHTTPStatus`, `TestParseMaxResults`, `Find` in key validation
- `tests/storage/config_test.go` — `MaxListSize` default, env override, cap, merge tests

## Patterns Established

- **Direct handler on `pkg/` interface**: When endpoints are pure pass-through to a `pkg/`-level system (no domain logic), the handler lives in `internal/api/` without a domain system layer
- **Wildcard path parameters for storage keys**: `/{key...}` captures slash-containing blob keys
- **Config-driven defaults with hard cap constant**: `MaxListSize` is configurable but capped at `MaxListCap`; `ParseMaxResults` encapsulates the parsing logic
- **`MapHTTPStatus` on storage errors**: Error-to-status mapping co-located with sentinel errors in `pkg/storage/errors.go`

## Validation Results

- `go vet ./...` passes
- `go test ./tests/...` — all 15 packages pass
- Manual verification against Azurite: list, find, and download all return correct responses
