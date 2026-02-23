# Issue #17 — Document HTTP Handlers, OpenAPI Specs, and API Wiring

## Context

Issue #16 delivered the document domain core (types, System interface, repository, mapping, errors). This task builds the HTTP presentation layer on top: handler struct with 6 endpoints, multipart upload processing, pdfcpu integration for PDF page count, and wiring into the API module. At completion, Herald is a fully functional document management API.

**OpenAPI note:** OpenAPI schema creation (`internal/documents/openapi.go`) is an AI responsibility handled after plan approval, before the implementation guide is written. It is not part of the guide.

## Implementation

### Step 1: Create `pkg/formatting/bytes.go`

**New file.** Reusable bidirectional byte size formatting package.

Unit table (base-1024): `B, KB, MB, GB, TB, PB, EB, ZB, YB`.

**Exported API:**

- `ParseBytes(s string) (int64, error)` — parses human-readable string like `"50MB"` into byte count. Splits into numeric prefix + unit suffix (case-insensitive), supports optional space. Returns error for unknown units or invalid numbers.
- `FormatBytes(n int64, precision int) string` — converts byte count to human-readable string using `math.Log`/`math.Pow` with base 1024.

### Step 2: Add `MaxUploadSize` to APIConfig

**File:** `internal/config/api.go`

Add `MaxUploadSize string` field (TOML: `max_upload_size`, default `"50MB"`, env var `HERALD_API_MAX_UPLOAD_SIZE`). Add `MaxUploadSizeBytes() int64` method that calls `formatting.ParseBytes`. Wire into `loadDefaults()`, `loadEnv()`, and `Merge()`.

### Step 3: Add `pdfcpu` dependency

```bash
go get github.com/pdfcpu/pdfcpu
```

### Step 4: Create `internal/documents/handler.go`

**New file.** Handler struct holding `sys System`, `logger *slog.Logger`, `pagination pagination.Config`, `maxUploadSize int64`.

Constructor: `NewHandler(sys, logger, pagination, maxUploadSize)`

`Routes() routes.Group` returns a Group with prefix `/documents`, tag `"Documents"`, schemas from `Spec.Schemas()`, and these routes:

| Method | Pattern | Handler | OpenAPI |
|--------|---------|---------|---------|
| GET | `` | `h.List` | `Spec.List` |
| GET | `/{id}` | `h.Find` | `Spec.Find` |
| POST | `` | `h.Upload` | `Spec.Upload` |
| POST | `/batch` | `h.BatchUpload` | `Spec.BatchUpload` |
| POST | `/search` | `h.Search` | `Spec.Search` |
| DELETE | `/{id}` | `h.Delete` | `Spec.Delete` |

**Handler methods:**

- **List** — `PageRequestFromQuery` + `FiltersFromQuery` → `sys.List` → `RespondJSON(200)`
- **Find** — parse `{id}` path param as UUID → `sys.Find` → `RespondJSON(200)` or `MapHTTPStatus` error
- **Upload** — `ParseMultipartForm(maxUploadSize)`, `FormFile("file")`, read bytes, `detectContentType`, `extractPDFPageCount` if PDF. Parse `external_id` (int) and `external_platform` (string) from form values. Build `CreateCommand` → `sys.Create` → `RespondJSON(201)`
- **BatchUpload** — `ParseMultipartForm(maxUploadSize)`, iterate `MultipartForm.File["files"]`. Parse parallel form arrays `external_id` and `external_platform` (matched positionally to files). Per-file: content type detection, PDF page count extraction. Build `[]CreateCommand` → `sys.CreateBatch` → `RespondJSON(201)`
- **Search** — decode JSON body into `SearchRequest` (embeds `PageRequest` + `Filters`) → normalize pagination → `sys.List` → `RespondJSON(200)`
- **Delete** — parse `{id}` UUID → `sys.Delete` → `w.WriteHeader(204)`

**Helper functions (unexported):**

- `extractPDFPageCount(data []byte) *int` — uses pdfcpu on a bytes reader. Non-fatal: returns nil on error (log warning).
- `detectContentType(header string, data []byte) string` — prefer multipart header if non-empty and not `application/octet-stream`; otherwise `http.DetectContentType(data)`.

**SearchRequest type** (in handler.go):

```go
type SearchRequest struct {
    pagination.PageRequest
    Filters
}
```

### Step 5: Add `Handler()` to System interface and wire API routes

**File:** `internal/documents/system.go` — add `Handler(maxUploadSize int64) *Handler` to the interface.

**File:** `internal/documents/repository.go` — implement on `repo` (already holds logger and pagination):

```go
func (r *repo) Handler(maxUploadSize int64) *Handler {
    return NewHandler(r, r.logger, r.pagination, maxUploadSize)
}
```

**File:** `internal/api/routes.go` — implement `registerRoutes`:

```go
routes.Register(mux, cfg.API.BasePath, spec,
    domain.Documents.Handler(cfg.API.MaxUploadSizeBytes()).Routes(),
)
```

### Step 6: Run `go mod tidy`

## Files Summary

| File | Action |
|------|--------|
| `pkg/formatting/bytes.go` | **Create** — ParseBytes + FormatBytes with B through YB |
| `internal/config/api.go` | Modify — add MaxUploadSize field, use formatting.ParseBytes |
| `internal/documents/handler.go` | **Create** — Handler struct, 6 endpoints, helpers |
| `internal/documents/system.go` | Modify — add `Handler()` to interface |
| `internal/documents/repository.go` | Modify — add `Handler()` method on repo |
| `internal/api/routes.go` | Modify — implement `registerRoutes` |
| `internal/documents/openapi.go` | **Create** — AI responsibility, pre-guide |
| `go.mod` / `go.sum` | Modified by `go get` / `go mod tidy` |

## Validation Criteria

- [ ] `go vet ./...` passes
- [ ] `go build ./cmd/server/` compiles cleanly
- [ ] MaxUploadSize configurable via `HERALD_API_MAX_UPLOAD_SIZE` env var
- [ ] All 6 endpoints defined with correct HTTP methods and patterns
- [ ] Upload endpoints require `external_id` and `external_platform` form fields
- [ ] Batch upload uses parallel form arrays for per-file metadata
- [ ] PDF page count extraction uses pdfcpu (non-fatal on failure)
- [ ] Batch upload handles multiple files with partial success semantics
- [ ] Search endpoint accepts JSON body with pagination + filters
- [ ] `formatting.ParseBytes` / `formatting.FormatBytes` round-trip correctly
- [ ] OpenAPI spec at `/api/openapi.json` includes all document endpoints and schemas
