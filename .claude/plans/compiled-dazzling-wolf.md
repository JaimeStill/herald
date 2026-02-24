# Issue #21 — Blob Storage Query Endpoints

## Context

The document domain provides CRUD operations against PostgreSQL, but there are no endpoints for querying Azure Blob Storage directly. Clients cannot list what's in the store, inspect blob metadata, or download files without going through the SQL layer. This adds three read-only endpoints under `/api/storage`.

## Architecture Approach

**No domain system needed.** The storage handler talks directly to `storage.System` (a `pkg/`-level interface). A domain system would be an unnecessary indirection for pure pass-through queries.

**Handler in `internal/api/`**, not a separate package. The `storageHandler` is unexported in the `api` package — an internal implementation detail of the API module wired alongside the documents handler.

**Marker-based pagination** — Azure Blob Storage uses opaque continuation tokens. The `ListResult` type does NOT use `PageRequest/PageResult` from `pkg/pagination/` because that pattern assumes offset-based pagination with total counts.

**Richer `Download` return type** — The current `Download` method returns `(io.ReadCloser, error)`. Since it's unused outside tests (which discard the result), we change it to return `(*DownloadResult, error)` where `DownloadResult` bundles the body stream with content type, content length, and content disposition metadata. This lets the download handler set proper HTTP headers from a single Azure `DownloadStream` call instead of making a separate `GetProperties` round-trip.

## Implementation

### Step 1: Add types and extend `System` interface

**File:** `pkg/storage/storage.go` (modify)

Add `time` to imports. Add four new types:

- `BlobItem` — name, content type, content length, last modified (for list results)
- `BlobProperties` — extends BlobItem with ETag and created-at (for properties endpoint)
- `ListResult` — items slice + optional next marker
- `DownloadResult` — body `io.ReadCloser`, content type, content length (replaces raw `io.ReadCloser` return)

Change `Download` return type from `(io.ReadCloser, error)` to `(*DownloadResult, error)`.

Add two new methods to `System`:

- `List(ctx, prefix, marker string, maxResults int32) (*ListResult, error)`
- `GetProperties(ctx, key string) (*BlobProperties, error)`

### Step 2: Update `Download` implementation

**File:** `pkg/storage/storage.go` (modify)

Update the `azure.Download` method to return `*DownloadResult` populated from the `DownloadStream` response's `ContentType` and `ContentLength` fields alongside the body.

### Step 3: Implement `List` on `azure` struct

**File:** `pkg/storage/storage.go` (modify)

Uses `a.client.NewListBlobsFlatPager(a.container, opts)` with `Prefix`, `Marker`, and `MaxResults` on `ListBlobsFlatOptions`. Calls `pager.NextPage(ctx)` once for a single page. Maps `resp.Segment.BlobItems` to `[]BlobItem`. Returns `NextMarker` if present.

Add pointer helpers `deref(*string) string` and `derefInt64(*int64) int64`.

### Step 4: Implement `GetProperties` on `azure` struct

**File:** `pkg/storage/storage.go` (modify)

Follows the existing `Exists` pattern — builds blob client via `a.client.ServiceClient().NewContainerClient(a.container).NewBlobClient(key)`, calls `GetProperties`, maps `BlobNotFound` to `ErrNotFound`. Returns `BlobProperties` with all available metadata.

### Step 5: Create storage handler

**File:** `internal/api/storage.go` (new)

Unexported `storageHandler` struct with `store storage.System` and `logger *slog.Logger`. Constructor `newStorageHandler`. Method `routes()` returning `routes.Group{Prefix: "/storage", ...}` with three routes:

- `GET ""` → `list` — parses `prefix`, `marker`, `max_results` query params, defaults max to 50, caps at 5000
- `GET "/properties"` → `properties` — parses `key` query param, calls `GetProperties`, maps errors
- `GET "/download"` → `download` — parses `key`, calls `Download` (now returns `DownloadResult` with metadata), streams body with `Content-Type`, `Content-Length`, `Content-Disposition` headers

Local `mapStorageHTTPStatus(err) int` for error → HTTP status mapping.

### Step 6: Wire into API module

**File:** `internal/api/routes.go` (modify)

Add `runtime *Runtime` parameter to `registerRoutes`. Create `storageHandler` from `runtime.Storage` and `runtime.Logger`. Register its route group alongside documents.

**File:** `internal/api/api.go` (modify)

Pass `runtime` to `registerRoutes`.

### Step 7: Update existing test

**File:** `tests/storage/storage_test.go` (modify)

Update the `Download` call in `TestKeyValidation` to match the new return type (`*DownloadResult` instead of `io.ReadCloser`). The test discards the result — only the variable type changes.

## Files Changed

| File | Status | Purpose |
|------|--------|---------|
| `pkg/storage/storage.go` | Modify | Types, interface change, new methods, azure implementations, helpers |
| `internal/api/storage.go` | New | Storage handler with list, properties, download endpoints |
| `internal/api/routes.go` | Modify | Wire storage handler, add runtime parameter |
| `internal/api/api.go` | Modify | Pass runtime to registerRoutes |
| `tests/storage/storage_test.go` | Modify | Update Download call for new return type |

## Validation Criteria

- [ ] `storage.System` interface extended with `List`, `GetProperties`, and updated `Download`
- [ ] `GET /api/storage` returns blob listing with marker-based pagination
- [ ] `GET /api/storage/properties?key=...` returns blob metadata JSON
- [ ] `GET /api/storage/download?key=...` streams blob with correct Content-Type and Content-Disposition headers
- [ ] All endpoints work against Azurite in local development
- [ ] `go vet ./...` passes
- [ ] `mise run test` passes
- [ ] API Cartographer documentation generated at `_project/api/storage/`
