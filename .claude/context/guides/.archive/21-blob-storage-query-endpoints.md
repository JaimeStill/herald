# 21 - Blob Storage Query Endpoints

## Problem Context

The document domain provides CRUD operations against PostgreSQL, but there are no endpoints for querying Azure Blob Storage directly. Clients cannot list what's in the store, inspect blob metadata, or download files without going through the SQL layer. This task adds three read-only HTTP endpoints under `/api/storage`.

## Architecture Approach

- **No domain system** — the storage handler talks directly to `storage.System`
- **Handler in `internal/api/`** as an unexported `storageHandler`
- **Marker-based pagination** — uses Azure's opaque continuation tokens, not `PageRequest/PageResult`
- **Richer `Download` return** — change from `(io.ReadCloser, error)` to `(*BlobResult, error)` since the method is unused outside tests

## Implementation

### Step 1: Add `MaxListSize` to storage config

**File:** `pkg/storage/config.go`

Add `MaxListSize` field to the `Config` struct:

```go
type Config struct {
	ContainerName    string `toml:"container_name"`
	ConnectionString string `toml:"connection_string"`
	MaxListSize      int32  `toml:"max_list_size"`
}
```

Add the corresponding field to the `Env` struct:

```go
type Env struct {
	ContainerName    string
	ConnectionString string
	MaxListSize      string
}
```

Add the default in `loadDefaults()`:

```go
func (c *Config) loadDefaults() {
	if c.ContainerName == "" {
		c.ContainerName = "documents"
	}
	if c.MaxListSize == 0 {
		c.MaxListSize = 50
	}
}
```

Add the env var override in `loadEnv()`:

```go
if env.MaxListSize != "" {
	if v := os.Getenv(env.MaxListSize); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			c.MaxListSize = int32(n)
		}
	}
}
```

This requires adding `"strconv"` to the imports.

Add the merge overlay in `Merge()`:

```go
if overlay.MaxListSize != 0 {
	c.MaxListSize = overlay.MaxListSize
}
```

**File:** `internal/config/config.go`

Add the env var mapping to `storageEnv`:

```go
var storageEnv = &storage.Env{
	ContainerName:    "HERALD_STORAGE_CONTAINER_NAME",
	ConnectionString: "HERALD_STORAGE_CONNECTION_STRING",
	MaxListSize:      "HERALD_STORAGE_MAX_LIST_SIZE",
}
```

### Step 2: Add `MaxListCap` constant and `ParseMaxResults` function

**File:** `pkg/storage/storage.go`

Add a package-level constant for the Azure hard ceiling and a function that encapsulates parsing, validation, and capping:

```go
const MaxListCap int32 = 5000
```

```go
func ParseMaxResults(s string, fallback int32) (int32, error) {
	if s == "" {
		return fallback, nil
	}

	n, err := strconv.Atoi(s)
	if err != nil || n < 1 {
		return 0, fmt.Errorf("invalid max_results parameter")
	}

	return min(int32(n), MaxListCap), nil
}
```

This requires adding `"strconv"` to the imports.

**File:** `pkg/storage/config.go`

Cap `MaxListSize` at `MaxListCap` in `loadDefaults()` and `loadEnv()`:

In `loadDefaults()`, after setting the default:

```go
if c.MaxListSize == 0 {
	c.MaxListSize = 50
}
if c.MaxListSize > MaxListCap {
	c.MaxListSize = MaxListCap
}
```

In `loadEnv()`, after parsing the env var value, apply the same cap:

```go
if env.MaxListSize != "" {
	if v := os.Getenv(env.MaxListSize); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			c.MaxListSize = min(int32(n), MaxListCap)
		}
	}
}
```

### Step 3: Add types, extend interface, update `Download` signature

**File:** `pkg/storage/storage.go`

Add `time` and `"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"` to imports.

Add these types after the `System` interface. `BlobMeta` is the single metadata type used across all three endpoints — list, properties, and download:

```go
type BlobMeta struct {
	Name          string    `json:"name"`
	ContentType   string    `json:"content_type"`
	ContentLength int64     `json:"content_length"`
	LastModified  time.Time `json:"last_modified"`
	ETag          string    `json:"etag"`
	CreatedAt     time.Time `json:"created_at"`
}

type BlobList struct {
	Blobs      []BlobMeta `json:"blobs"`
	NextMarker string     `json:"next_marker,omitempty"`
}

type BlobResult struct {
	BlobMeta
	Body io.ReadCloser `json:"-"`
}
```

Update `Download` in the `System` interface from:

```go
Download(ctx context.Context, key string) (io.ReadCloser, error)
```

to:

```go
Download(ctx context.Context, key string) (*BlobResult, error)
```

Add two new methods to `System`:

```go
List(ctx context.Context, prefix string, marker string, maxResults int32) (*BlobList, error)
Find(ctx context.Context, key string) (*BlobMeta, error)
```

### Step 4: Update `Download` implementation and add new methods

**File:** `pkg/storage/storage.go`

Replace the existing `Download` method:

```go
func (a *azure) Download(ctx context.Context, key string) (*BlobResult, error) {
	if err := validateKey(key); err != nil {
		return nil, err
	}

	resp, err := a.client.DownloadStream(ctx, a.container, key, nil)
	if err != nil {
		if bloberror.HasCode(err, bloberror.BlobNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("download blob %s: %w", key, err)
	}

	result := &BlobResult{
		BlobMeta: BlobMeta{Name: key},
		Body:     resp.Body,
	}
	if resp.ContentType != nil {
		result.ContentType = *resp.ContentType
	}
	if resp.ContentLength != nil {
		result.ContentLength = *resp.ContentLength
	}
	return result, nil
}
```

Add the `List` method. `NewListBlobsFlatPager` is on the container client, not the top-level azblob client:

```go
func (a *azure) List(ctx context.Context, prefix string, marker string, maxResults int32) (*BlobList, error) {
	containerClient := a.client.ServiceClient().NewContainerClient(a.container)

	opts := &container.ListBlobsFlatOptions{
		MaxResults: &maxResults,
	}
	if prefix != "" {
		opts.Prefix = &prefix
	}
	if marker != "" {
		opts.Marker = &marker
	}

	pager := containerClient.NewListBlobsFlatPager(opts)
	if !pager.More() {
		return &BlobList{Blobs: []BlobMeta{}}, nil
	}

	resp, err := pager.NextPage(ctx)
	if err != nil {
		return nil, fmt.Errorf("list blobs: %w", err)
	}

	blobs := make([]BlobMeta, 0, len(resp.Segment.BlobItems))
	for _, b := range resp.Segment.BlobItems {
		var meta BlobMeta
		if b.Name != nil {
			meta.Name = *b.Name
		}
		if b.Properties != nil {
			if b.Properties.ContentType != nil {
				meta.ContentType = *b.Properties.ContentType
			}
			if b.Properties.ContentLength != nil {
				meta.ContentLength = *b.Properties.ContentLength
			}
			if b.Properties.LastModified != nil {
				meta.LastModified = *b.Properties.LastModified
			}
			if b.Properties.ETag != nil {
				meta.ETag = string(*b.Properties.ETag)
			}
			if b.Properties.CreationTime != nil {
				meta.CreatedAt = *b.Properties.CreationTime
			}
		}
		blobs = append(blobs, meta)
	}

	result := &BlobList{Blobs: blobs}
	if resp.NextMarker != nil && *resp.NextMarker != "" {
		result.NextMarker = *resp.NextMarker
	}
	return result, nil
}
```

Add `Find` following the existing `Exists` pattern:

```go
func (a *azure) Find(ctx context.Context, key string) (*BlobMeta, error) {
	if err := validateKey(key); err != nil {
		return nil, err
	}

	blobClient := a.client.
		ServiceClient().
		NewContainerClient(a.container).
		NewBlobClient(key)

	resp, err := blobClient.GetProperties(ctx, nil)
	if err != nil {
		if bloberror.HasCode(err, bloberror.BlobNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get blob properties %s: %w", key, err)
	}

	meta := &BlobMeta{Name: key}
	if resp.ContentType != nil {
		meta.ContentType = *resp.ContentType
	}
	if resp.ContentLength != nil {
		meta.ContentLength = *resp.ContentLength
	}
	if resp.LastModified != nil {
		meta.LastModified = *resp.LastModified
	}
	if resp.ETag != nil {
		meta.ETag = string(*resp.ETag)
	}
	if resp.CreationTime != nil {
		meta.CreatedAt = *resp.CreationTime
	}
	return meta, nil
}
```

### Step 5: Create storage handler

**File:** `internal/api/storage.go` (new)

```go
package api

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path"
	"strconv"

	"github.com/JaimeStill/herald/pkg/handlers"
	"github.com/JaimeStill/herald/pkg/routes"
	"github.com/JaimeStill/herald/pkg/storage"
)

type storageHandler struct {
	store       storage.System
	logger      *slog.Logger
	maxListSize int32
}

func newStorageHandler(store storage.System, logger *slog.Logger, maxListSize int32) *storageHandler {
	return &storageHandler{
		store:       store,
		logger:      logger.With("handler", "storage"),
		maxListSize: maxListSize,
	}
}

func (h *storageHandler) routes() routes.Group {
	return routes.Group{
		Prefix: "/storage",
		Routes: []routes.Route{
			{Method: "GET", Pattern: "", Handler: h.list},
			{Method: "GET", Pattern: "/download/{key...}", Handler: h.download},
			{Method: "GET", Pattern: "/{key...}", Handler: h.find},
		},
	}
}

func (h *storageHandler) list(w http.ResponseWriter, r *http.Request) {
	prefix := r.URL.Query().Get("prefix")
	marker := r.URL.Query().Get("marker")

	maxResults, err := storage.ParseMaxResults(r.URL.Query().Get("max_results"), h.maxListSize)
	if err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, err)
		return
	}

	result, err := h.store.List(r.Context(), prefix, marker, maxResults)
	if err != nil {
		handlers.RespondError(w, h.logger, http.StatusInternalServerError, err)
		return
	}

	handlers.RespondJSON(w, http.StatusOK, result)
}

func (h *storageHandler) find(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")

	meta, err := h.store.Find(r.Context(), key)
	if err != nil {
		handlers.RespondError(w, h.logger, storage.MapHTTPStatus(err), err)
		return
	}

	handlers.RespondJSON(w, http.StatusOK, meta)
}

func (h *storageHandler) download(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")

	result, err := h.store.Download(r.Context(), key)
	if err != nil {
		handlers.RespondError(w, h.logger, storage.MapHTTPStatus(err), err)
		return
	}
	defer result.Body.Close()

	w.Header().Set("Content-Type", result.ContentType)
	if result.ContentLength > 0 {
		w.Header().Set("Content-Length", strconv.FormatInt(result.ContentLength, 10))
	}
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", path.Base(key)))
	w.WriteHeader(http.StatusOK)
	io.Copy(w, result.Body)
}

// storage.MapHTTPStatus removed — use storage.MapHTTPStatus from pkg/storage/errors.go
```

### Step 6: Wire into API module

**File:** `internal/api/routes.go`

Add `runtime *Runtime` parameter and register the storage handler route group:

```go
func registerRoutes(
	mux *http.ServeMux,
	domain *Domain,
	cfg *config.Config,
	runtime *Runtime,
) {
	storageH := newStorageHandler(runtime.Storage, runtime.Logger, cfg.Storage.MaxListSize)

	routes.Register(
		mux,
		domain.Documents.Handler(cfg.API.MaxUploadSizeBytes()).Routes(),
		storageH.routes(),
	)
}
```

**File:** `internal/api/api.go`

Update the `registerRoutes` call to pass `runtime`:

```go
registerRoutes(mux, domain, cfg, runtime)
```

### Step 7: Update existing test

**File:** `tests/storage/storage_test.go`

The `Download` call in `TestKeyValidation` currently returns `(io.ReadCloser, error)` — now returns `(*storage.BlobResult, error)`. Since the result is discarded with `_`, no other changes needed.

## Validation Criteria

- [ ] `storage.System` interface extended with `List`, `Find`, and updated `Download`
- [ ] `GET /api/storage` returns blob listing with marker-based pagination
- [ ] `GET /api/storage/properties?key=...` returns blob metadata JSON
- [ ] `GET /api/storage/download?key=...` streams blob with correct Content-Type and Content-Disposition headers
- [ ] All endpoints work against Azurite in local development
- [ ] `go vet ./...` passes
- [ ] Tests pass (`mise run test`)
- [ ] API Cartographer documentation generated at `_project/api/storage/`
