# 124 - Make AgentScope Configurable for Azure Government

## Problem Context

`AgentScope` is hardcoded as `https://cognitiveservices.azure.com/.default` in `pkg/auth/config.go`. Azure Government uses `https://cognitiveservices.azure.us/.default`. Similarly, `TokenScope` is hardcoded in `pkg/database/database.go`. While the database scope is currently universal across clouds, making it configurable provides the same escape hatch — patchable via config if the scope ever changes, without a code release.

## Architecture Approach

Follow the established config patterns in `auth.Config` and `database.Config`: add struct fields with JSON tags, set defaults in `loadDefaults()`, override via `loadEnv()`, and include in `Merge()`. Constants are removed and consumers read scopes from config instead.

## Implementation

### Step 1: Convert AgentScope to configurable field (`pkg/auth/config.go`)

Remove the `AgentScope` constant and add it as a field on `Config` and `Env`:

```go
// Remove this constant:
// const AgentScope = "https://cognitiveservices.azure.com/.default"
```

Add `AgentScope` field to `Config` (after `CacheLocation`):

```go
type Config struct {
	Mode            Mode          `json:"auth_mode"`
	ManagedIdentity bool          `json:"managed_identity"`
	TenantID        string        `json:"tenant_id"`
	ClientID        string        `json:"client_id"`
	ClientSecret    string        `json:"client_secret"`
	Authority       string        `json:"authority"`
	Scope           string        `json:"scope"`
	CacheLocation   CacheLocation `json:"cache_location"`
	AgentScope      string        `json:"agent_scope"`
}
```

Add `AgentScope` field to `Env` (after `CacheLocation`):

```go
type Env struct {
	Mode            string
	ManagedIdentity string
	TenantID        string
	ClientID        string
	ClientSecret    string
	Authority       string
	Scope           string
	CacheLocation   string
	AgentScope      string
}
```

Add default in `loadDefaults()` (after `CacheLocation` default):

```go
if c.AgentScope == "" {
	c.AgentScope = "https://cognitiveservices.azure.com/.default"
}
```

Add env override in `loadEnv()` (after `CacheLocation` block):

```go
if env.AgentScope != "" {
	if v := os.Getenv(env.AgentScope); v != "" {
		c.AgentScope = v
	}
}
```

Add merge logic in `Merge()` (after `CacheLocation` block):

```go
if overlay.AgentScope != "" {
	c.AgentScope = overlay.AgentScope
}
```

### Step 2: Add env var mapping (`internal/config/config.go`)

Add `AgentScope` to the `authEnv` var (after `CacheLocation`):

```go
var authEnv = &auth.Env{
	Mode:            "HERALD_AUTH_MODE",
	ManagedIdentity: "HERALD_AUTH_MANAGED_IDENTITY",
	TenantID:        "HERALD_AUTH_TENANT_ID",
	ClientID:        "HERALD_AUTH_CLIENT_ID",
	ClientSecret:    "HERALD_AUTH_CLIENT_SECRET",
	Authority:       "HERALD_AUTH_AUTHORITY",
	Scope:           "HERALD_AUTH_SCOPE",
	CacheLocation:   "HERALD_AUTH_CACHE_LOCATION",
	AgentScope:      "HERALD_AUTH_AGENT_SCOPE",
}
```

### Step 3: Read scope from config (`internal/infrastructure/infrastructure.go`)

Update `newAgentFactory` to accept the scope as a parameter instead of referencing the removed constant.

Change the function signature:

```go
func newAgentFactory(
	agentCfg gaconfig.AgentConfig,
	cred azcore.TokenCredential,
	managedIdentity bool,
	agentScope string,
) func(ctx context.Context) (agent.Agent, error) {
```

Replace `auth.AgentScope` with the `agentScope` parameter in `GetToken`:

```go
tok, err := cred.GetToken(ctx, policy.TokenRequestOptions{
	Scopes: []string{agentScope},
})
```

Update the call site in `New()`:

```go
newAgent := newAgentFactory(cfg.Agent, cred, cfg.Auth.ManagedIdentity, cfg.Auth.AgentScope)
```

Remove the `"github.com/JaimeStill/herald/pkg/auth"` import (no longer needed in this file).

### Step 4: Convert TokenScope to configurable field (`pkg/database/config.go`)

Add `TokenScope` field to `Config` (after `TokenLifetime`):

```go
type Config struct {
	Host            string `json:"host"`
	Port            int    `json:"port"`
	Name            string `json:"name"`
	User            string `json:"user"`
	Password        string `json:"password"`
	SSLMode         string `json:"ssl_mode"`
	MaxOpenConns    int    `json:"max_open_conns"`
	MaxIdleConns    int    `json:"max_idle_conns"`
	ConnMaxLifetime string `json:"conn_max_lifetime"`
	ConnTimeout     string `json:"conn_timeout"`
	TokenLifetime   string `json:"token_lifetime"`
	TokenScope      string `json:"token_scope"`
}
```

Add `TokenScope` field to `Env` (after `TokenLifetime`):

```go
type Env struct {
	Host            string
	Port            string
	Name            string
	User            string
	Password        string
	SSLMode         string
	MaxOpenConns    string
	MaxIdleConns    string
	ConnMaxLifetime string
	ConnTimeout     string
	TokenLifetime   string
	TokenScope      string
}
```

Add default in `loadDefaults()` (after `TokenLifetime` default):

```go
if c.TokenScope == "" {
	c.TokenScope = "https://ossrdbms-aad.database.windows.net/.default"
}
```

Add env override in `loadEnv()` (after `TokenLifetime` block):

```go
if env.TokenScope != "" {
	if v := os.Getenv(env.TokenScope); v != "" {
		c.TokenScope = v
	}
}
```

Add merge logic in `Merge()` (after `TokenLifetime` block):

```go
if overlay.TokenScope != "" {
	c.TokenScope = overlay.TokenScope
}
```

### Step 5: Add env var mapping (`internal/config/config.go`)

Add `TokenScope` to the `databaseEnv` var (after `TokenLifetime`):

```go
var databaseEnv = &database.Env{
	Host:            "HERALD_DB_HOST",
	Port:            "HERALD_DB_PORT",
	Name:            "HERALD_DB_NAME",
	User:            "HERALD_DB_USER",
	Password:        "HERALD_DB_PASSWORD",
	SSLMode:         "HERALD_DB_SSL_MODE",
	MaxOpenConns:    "HERALD_DB_MAX_OPEN_CONNS",
	MaxIdleConns:    "HERALD_DB_MAX_IDLE_CONNS",
	ConnMaxLifetime: "HERALD_DB_CONN_MAX_LIFETIME",
	ConnTimeout:     "HERALD_DB_CONN_TIMEOUT",
	TokenLifetime:   "HERALD_DB_TOKEN_LIFETIME",
	TokenScope:      "HERALD_DB_TOKEN_SCOPE",
}
```

### Step 6: Read scope from config (`pkg/database/database.go`)

Remove the `TokenScope` constant:

```go
// Remove this constant:
// const TokenScope = "https://ossrdbms-aad.database.windows.net/.default"
```

Replace `TokenScope` with `cfg.TokenScope` in `NewWithCredential`:

```go
token, err := cred.GetToken(ctx, policy.TokenRequestOptions{
	Scopes: []string{cfg.TokenScope},
})
```

The `cfg` variable is already available — `NewWithCredential` receives `cfg *Config` as its first parameter.

## Validation Criteria

- [ ] `go vet ./...` passes
- [ ] `go build ./cmd/server/` succeeds
- [ ] `go test ./tests/...` passes
- [ ] No references to `auth.AgentScope` or `database.TokenScope` constants remain in the codebase
- [ ] Default behavior unchanged when env vars are not set
