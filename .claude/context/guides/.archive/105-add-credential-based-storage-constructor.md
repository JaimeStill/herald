# 105 — Add credential-based storage constructor

## Problem Context

The `pkg/storage/` package only supports connection string auth via `New()`. Managed identity (Objective #97) requires creating the Azure Blob client with `azblob.NewClient(serviceURL, cred, nil)`. This task adds a `NewWithCredential` constructor and relaxes config validation so each constructor validates its own requirements.

## Architecture Approach

Dual-constructor pattern: `New` for connection strings, `NewWithCredential` for managed identity credentials. Both produce the same `azure` struct — only client construction differs. Config validation is split: `validate()` checks universal invariants (ContainerName, MaxListSize), each constructor guards its own required fields.

## Implementation

### Step 1: Add `ServiceURL` to storage config

**`pkg/storage/config.go`**

Add `ServiceURL` field to `Config`:

```go
type Config struct {
	ContainerName    string `json:"container_name"`
	ConnectionString string `json:"connection_string"`
	ServiceURL       string `json:"service_url"`
	MaxListSize      int32  `json:"max_list_size"`
}
```

Add `ServiceURL` to `Env`:

```go
type Env struct {
	ContainerName    string
	ConnectionString string
	ServiceURL       string
	MaxListSize      string
}
```

Add `ServiceURL` to `Merge` (after the `ConnectionString` block):

```go
if overlay.ServiceURL != "" {
	c.ServiceURL = overlay.ServiceURL
}
```

Add `ServiceURL` to `loadEnv` (after the `ConnectionString` block):

```go
if env.ServiceURL != "" {
	if v := os.Getenv(env.ServiceURL); v != "" {
		c.ServiceURL = v
	}
}
```

Relax `validate()` — remove `ConnectionString` requirement:

```go
func (c *Config) validate() error {
	if c.ContainerName == "" {
		return fmt.Errorf("container_name required")
	}
	return nil
}
```

### Step 2: Add `NewWithCredential` constructor and guard `New`

**`pkg/storage/storage.go`**

Add `azcore` import:

```go
"github.com/Azure/azure-sdk-for-go/sdk/azcore"
```

Add connection string guard at the top of existing `New`:

```go
func New(cfg *Config, logger *slog.Logger) (System, error) {
	if cfg.ConnectionString == "" {
		return nil, fmt.Errorf("connection_string required for connection string auth")
	}

	client, err := azblob.NewClientFromConnectionString(cfg.ConnectionString, nil)
	// ... rest unchanged
```

Add `NewWithCredential` constructor after `New`:

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

### Step 3: Add `ServiceURL` env var mapping

**`internal/config/config.go`**

Update `storageEnv`:

```go
var storageEnv = &storage.Env{
	ContainerName:    "HERALD_STORAGE_CONTAINER_NAME",
	ConnectionString: "HERALD_STORAGE_CONNECTION_STRING",
	ServiceURL:       "HERALD_STORAGE_SERVICE_URL",
	MaxListSize:      "HERALD_STORAGE_MAX_LIST_SIZE",
}
```

## Validation Criteria

- [ ] `storage.Config` has `ServiceURL` field with JSON tag, env override, and merge support
- [ ] `validate()` only checks `ContainerName` (universal invariant)
- [ ] `New()` guards `ConnectionString` internally before creating the client
- [ ] `NewWithCredential()` guards `ServiceURL`, accepts `azcore.TokenCredential`, creates client via `azblob.NewClient`
- [ ] `HERALD_STORAGE_SERVICE_URL` env var is mapped in `internal/config/config.go`
- [ ] `go vet ./...` passes
- [ ] `go build ./cmd/server/` succeeds
- [ ] `go test ./tests/...` passes
