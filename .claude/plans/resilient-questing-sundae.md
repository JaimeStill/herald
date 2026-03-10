# 105 — Add credential-based storage constructor

## Context

Issue #105, part of Objective #97 (Managed Identity for Azure Services). The `pkg/storage/` package currently only supports connection string auth via `New()`. Azure managed identity requires creating the client with `azblob.NewClient(serviceURL, cred, nil)` instead of `azblob.NewClientFromConnectionString()`. This task adds a `NewWithCredential` constructor and relaxes config validation so each constructor validates its own requirements.

## Plan

### 1. Update `pkg/storage/config.go`

**Add `ServiceURL` field to `Config`:**
```go
type Config struct {
    ContainerName    string `json:"container_name"`
    ConnectionString string `json:"connection_string"`
    ServiceURL       string `json:"service_url"`
    MaxListSize      int32  `json:"max_list_size"`
}
```

**Add `ServiceURL` to `Env`:**
```go
type Env struct {
    ContainerName    string
    ConnectionString string
    ServiceURL       string
    MaxListSize      string
}
```

**Add `ServiceURL` to `Merge`:**
```go
if overlay.ServiceURL != "" {
    c.ServiceURL = overlay.ServiceURL
}
```

**Add `ServiceURL` to `loadEnv`:**
```go
if env.ServiceURL != "" {
    if v := os.Getenv(env.ServiceURL); v != "" {
        c.ServiceURL = v
    }
}
```

**Relax `validate()`** — remove `ConnectionString` requirement, keep only universal invariants:
```go
func (c *Config) validate() error {
    if c.ContainerName == "" {
        return fmt.Errorf("container_name required")
    }
    return nil
}
```

### 2. Update `pkg/storage/storage.go`

**Add `NewWithCredential` constructor:**
```go
func NewWithCredential(cfg *Config, cred azcore.TokenCredential, logger *slog.Logger) (System, error) {
    if cfg.ServiceURL == "" {
        return nil, fmt.Errorf("service_url required for credential auth")
    }

    client, err := azblob.NewClient(cfg.ServiceURL, cred, nil)
    if err != nil {
        return nil, fmt.Errorf("create storage client: %w", err)
    }

    return &azure{
        client:    client,
        container: cfg.ContainerName,
        logger:    logger.With("system", "storage"),
    }, nil
}
```

**Update `New` to validate `ConnectionString` internally:**
```go
func New(cfg *Config, logger *slog.Logger) (System, error) {
    if cfg.ConnectionString == "" {
        return nil, fmt.Errorf("connection_string required for connection string auth")
    }
    // ... rest unchanged
}
```

New import: `"github.com/Azure/azure-sdk-for-go/sdk/azcore"`

### 3. Update `internal/config/config.go`

Add `ServiceURL` env var to `storageEnv`:
```go
var storageEnv = &storage.Env{
    ContainerName:    "HERALD_STORAGE_CONTAINER_NAME",
    ConnectionString: "HERALD_STORAGE_CONNECTION_STRING",
    ServiceURL:       "HERALD_STORAGE_SERVICE_URL",
    MaxListSize:      "HERALD_STORAGE_MAX_LIST_SIZE",
}
```

### Files

| File | Change |
|------|--------|
| `pkg/storage/config.go` | Add `ServiceURL` field, env, merge, relax validate |
| `pkg/storage/storage.go` | Add `NewWithCredential`, guard `ConnectionString` in `New` |
| `internal/config/config.go` | Add `ServiceURL` env var mapping |

### Validation

- `go vet ./...` passes
- `go build ./cmd/server/` succeeds
- `go test ./tests/...` passes (existing tests unaffected)
