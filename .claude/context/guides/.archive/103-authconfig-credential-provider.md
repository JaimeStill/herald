# 103 - Add AuthConfig and credential provider infrastructure

## Problem Context

Objective #96 (Azure Identity Credential Infrastructure) requires a credential provider foundation so downstream objectives can authenticate to Azure services via managed identity (#97) and protect the HTTP API with Entra JWT tokens (#98). This task adds the `azidentity` SDK, creates `AuthConfig` with the three-phase finalize pattern, and wires credential creation into `infrastructure.New()`. The credential is optional — nil when `auth_mode` is `none` (default), preserving all current behavior.

## Architecture Approach

`AuthConfig` lives in `internal/config/` as a same-package config (like `ServerConfig`) with direct env constants. `AuthMode` is a typed `string` enum (`AuthModeNone`, `AuthModeAzure`) for compile-time safety. The credential factory is a method on `AuthConfig` — the factory is trivial and config owns the mode decision. When all three service principal fields are set (TenantID, ClientID, ClientSecret), the factory uses `NewClientSecretCredential`. Otherwise, it uses `NewDefaultAzureCredential` which walks the full Azure credential chain (managed identity, workload identity, Azure CLI). The `Credential` field on `Infrastructure` is `azcore.TokenCredential` (nil-safe for `none` mode).

## Implementation

### Step 1: Add `azidentity` dependency

```bash
go get github.com/Azure/azure-sdk-for-go/sdk/azidentity
```

### Step 2: Create `internal/config/auth.go`

New file:

```go
package config

import (
	"fmt"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
)

const (
	EnvAuthMode         = "HERALD_AUTH_MODE"
	EnvAuthTenantID     = "HERALD_AUTH_TENANT_ID"
	EnvAuthClientID     = "HERALD_AUTH_CLIENT_ID"
	EnvAuthClientSecret = "HERALD_AUTH_CLIENT_SECRET"
)

type AuthMode string

const (
	AuthModeNone  AuthMode = "none"
	AuthModeAzure AuthMode = "azure"
)

type AuthConfig struct {
	Mode         AuthMode `json:"auth_mode"`
	TenantID     string   `json:"tenant_id"`
	ClientID     string   `json:"client_id"`
	ClientSecret string   `json:"client_secret"`
}

func (c *AuthConfig) Finalize() error {
	c.loadDefaults()
	c.loadEnv()
	return c.validate()
}

func (c *AuthConfig) Merge(overlay *AuthConfig) {
	if overlay.Mode != "" {
		c.Mode = overlay.Mode
	}
	if overlay.TenantID != "" {
		c.TenantID = overlay.TenantID
	}
	if overlay.ClientID != "" {
		c.ClientID = overlay.ClientID
	}
	if overlay.ClientSecret != "" {
		c.ClientSecret = overlay.ClientSecret
	}
}

func (c *AuthConfig) TokenCredential() (azcore.TokenCredential, error) {
	switch c.Mode {
	case AuthModeNone:
		return nil, nil
	case AuthModeAzure:
		return c.azureCredential()
	default:
		return nil, fmt.Errorf("unsupported auth mode: %s", c.Mode)
	}
}

func (c *AuthConfig) azureCredential() (azcore.TokenCredential, error) {
	if c.TenantID != "" && c.ClientID != "" && c.ClientSecret != "" {
		cred, err := azidentity.NewClientSecretCredential(
			c.TenantID, c.ClientID, c.ClientSecret, nil,
		)
		if err != nil {
			return nil, fmt.Errorf("create client secret credential: %w", err)
		}
		return cred, nil
	}

	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, fmt.Errorf("create default azure credential: %w", err)
	}
	return cred, nil
}

func (c *AuthConfig) loadDefaults() {
	if c.Mode == "" {
		c.Mode = AuthModeNone
	}
}

func (c *AuthConfig) loadEnv() {
	if v := os.Getenv(EnvAuthMode); v != "" {
		c.Mode = AuthMode(v)
	}
	if v := os.Getenv(EnvAuthTenantID); v != "" {
		c.TenantID = v
	}
	if v := os.Getenv(EnvAuthClientID); v != "" {
		c.ClientID = v
	}
	if v := os.Getenv(EnvAuthClientSecret); v != "" {
		c.ClientSecret = v
	}
}

func (c *AuthConfig) validate() error {
	switch c.Mode {
	case AuthModeNone, AuthModeAzure:
		return nil
	default:
		return fmt.Errorf("invalid auth_mode %q: must be %q or %q", c.Mode, AuthModeNone, AuthModeAzure)
	}
}
```

### Step 3: Wire `AuthConfig` into root `Config`

In `internal/config/config.go`, add the `Auth` field to the `Config` struct:

```go
type Config struct {
	Agent           gaconfig.AgentConfig `json:"agent"`
	Auth            AuthConfig           `json:"auth"`
	Server          ServerConfig         `json:"server"`
	Database        database.Config      `json:"database"`
	Storage         storage.Config       `json:"storage"`
	API             APIConfig            `json:"api"`
	ShutdownTimeout string               `json:"shutdown_timeout"`
	Version         string               `json:"version"`
}
```

Add `c.Auth.Merge(&overlay.Auth)` to `Merge()`:

```go
func (c *Config) Merge(overlay *Config) {
	if overlay.ShutdownTimeout != "" {
		c.ShutdownTimeout = overlay.ShutdownTimeout
	}
	if overlay.Version != "" {
		c.Version = overlay.Version
	}
	c.Agent.Merge(&overlay.Agent)
	c.Auth.Merge(&overlay.Auth)
	c.Server.Merge(&overlay.Server)
	c.Database.Merge(&overlay.Database)
	c.Storage.Merge(&overlay.Storage)
	c.API.Merge(&overlay.API)
}
```

Add `c.Auth.Finalize()` to `finalize()` — before agent finalization:

```go
func (c *Config) finalize() error {
	c.loadDefaults()
	c.loadEnv()

	if err := c.validate(); err != nil {
		return err
	}
	if err := c.Auth.Finalize(); err != nil {
		return fmt.Errorf("auth: %w", err)
	}
	if err := FinalizeAgent(&c.Agent); err != nil {
		return fmt.Errorf("agent: %w", err)
	}
	// ... rest unchanged
}
```

### Step 4: Add `Credential` to `Infrastructure`

In `internal/infrastructure/infrastructure.go`, add the credential field and create it in `New()`.

Add import:

```go
"github.com/Azure/azure-sdk-for-go/sdk/azcore"
```

Add field to struct:

```go
type Infrastructure struct {
	Lifecycle  *lifecycle.Coordinator
	Logger     *slog.Logger
	Database   database.System
	Storage    storage.System
	Agent      gaconfig.AgentConfig
	Credential azcore.TokenCredential
}
```

In `New()`, create the credential after storage init (before agent validation):

```go
cred, err := cfg.Auth.TokenCredential()
if err != nil {
	return nil, fmt.Errorf("credential init failed: %w", err)
}
```

And include it in the return:

```go
return &Infrastructure{
	Lifecycle:  lc,
	Logger:     logger,
	Database:   db,
	Storage:    store,
	Agent:      cfg.Agent,
	Credential: cred,
}, nil
```

### Step 5: Propagate through `api.NewRuntime()`

In `internal/api/runtime.go`, add `Credential` to the `Infrastructure` literal:

```go
func NewRuntime(cfg *config.Config, infra *infrastructure.Infrastructure) *Runtime {
	return &Runtime{
		Infrastructure: &infrastructure.Infrastructure{
			Agent:      cfg.Agent,
			Credential: infra.Credential,
			Lifecycle:  infra.Lifecycle,
			Logger:     infra.Logger.With("module", "api"),
			Database:   infra.Database,
			Storage:    infra.Storage,
		},
		Pagination: cfg.API.Pagination,
	}
}
```

## Validation Criteria

- [ ] `go build ./...` succeeds with `azidentity` dependency
- [ ] `go vet ./...` passes
- [ ] `go test ./tests/...` passes
- [ ] Default config (no `auth` section): `Infrastructure.Credential` is nil, all existing behavior unchanged
- [ ] Config with `"auth": {"auth_mode": "azure"}`: `TokenCredential()` returns a valid credential
- [ ] Config with invalid `auth_mode` (e.g., `"foo"`): clear validation error
- [ ] `HERALD_AUTH_*` env vars override JSON config values
