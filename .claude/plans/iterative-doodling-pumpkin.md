# Issue #6 — Storage Abstraction

## Context

Herald needs a blob storage layer for persisting PDF documents in Azure Blob Storage. The `pkg/storage/` package already has `config.go` wired into `internal/config/`. This task adds the `System` interface and Azure Blob Storage implementation, following the same structural pattern as `pkg/database/` (interface + private struct + `New` constructor + `Start` lifecycle hooks).

Agent-lab's filesystem storage provides the architectural template (interface, lifecycle integration, error sentinels, infrastructure injection), but Herald's implementation differs in two key ways: streaming I/O (`io.Reader`/`io.ReadCloser` instead of `[]byte`) and Azure SDK instead of filesystem operations.

## Files

### New: `pkg/storage/errors.go`

Sentinel errors following `pkg/database/errors.go` pattern:

- `ErrNotFound` — blob does not exist (maps from `bloberror.BlobNotFound`)
- `ErrEmptyKey` — empty key string
- `ErrInvalidKey` — key contains path traversal (`..`)

### New: `pkg/storage/storage.go`

Single file containing interface, private struct, constructor, and all methods (mirrors `database.go`).

**Interface:**

```go
type System interface {
    Start(lc *lifecycle.Coordinator) error
    Upload(ctx context.Context, key string, reader io.Reader, contentType string) error
    Download(ctx context.Context, key string) (io.ReadCloser, error)
    Delete(ctx context.Context, key string) error
    Exists(ctx context.Context, key string) (bool, error)
}
```

**Private struct:**

```go
type azure struct {
    client        *azblob.Client
    containerName string
    logger        *slog.Logger
}
```

**Constructor** — `New(cfg *Config, logger *slog.Logger) (System, error)`:
- Creates `azblob.Client` from connection string (validates but makes no network calls, like `sql.Open`)
- Stores container name and logger

**Start** — registers lifecycle hooks:
- `OnStartup`: calls `CreateContainer` (idempotent — ignores `ContainerAlreadyExists`)
- `OnShutdown`: blocks on `<-lc.Context().Done()`, logs stop (azblob client is stateless, no cleanup needed)

**Upload** — validates key, calls `client.UploadStream` with content type in `HTTPHeaders`

**Download** — validates key, calls `client.DownloadStream`, returns `resp.Body` (caller closes); maps `BlobNotFound` → `ErrNotFound`

**Delete** — validates key, calls `client.DeleteBlob`; maps `BlobNotFound` → `ErrNotFound`

**Exists** — validates key, uses sub-client chain (`ServiceClient` → `NewContainerClient` → `NewBlobClient` → `GetProperties`); `BlobNotFound` → `(false, nil)`, success → `(true, nil)`

**validateKey** — unexported helper: rejects empty keys (`ErrEmptyKey`) and `..` segments (`ErrInvalidKey`)

### Dependency

```
go get github.com/Azure/azure-sdk-for-go/sdk/storage/azblob
```

Imports needed: `azblob`, `azblob/blob` (for `HTTPHeaders`), `azblob/bloberror` (for error code matching).

## Validation

- `go build ./...` passes
- `go vet ./...` passes
- `go test ./tests/...` passes (existing config tests + new storage tests)
- Integration test with Azurite: upload → exists → download+compare → delete → verify gone
