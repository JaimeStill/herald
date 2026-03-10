# 106 — Add Token-Based Database Authentication

## Problem Context

Part of Objective #97 (Managed Identity for Azure Services). Herald needs an alternate database constructor that authenticates to Azure PostgreSQL using Entra tokens instead of a static password. This enables managed identity deployments where no credentials are stored in configuration.

## Architecture Approach

Follow the dual-constructor pattern from `pkg/storage/storage.go` (#105). Add `NewWithCredential` alongside the existing `New` constructor. The new constructor uses pgx's `stdlib.OpenDB` with a `BeforeConnect` hook that acquires a fresh Entra token on every new connection, injecting it as the password.

A configurable `TokenLifetime` field on `Config` (default 45m) controls `ConnMaxLifetime` in credential mode, providing a safety net against Entra token expiry (~1 hour). This is separate from `ConnMaxLifetime`, which applies to connection-string auth via `New`.

The existing `New` constructor remains unchanged — it continues to use `sql.Open("pgx", dsn)` for connection-string auth.

## Implementation

### Step 1: Create `pkg/database/constants.go`

New file with the OAuth2 token scope for Azure Database for PostgreSQL:

```go
package database

const TokenScope = "https://ossrdbms-aad.database.windows.net/.default"
```

### Step 2: Add `TokenLifetime` to `pkg/database/config.go`

Add the `TokenLifetime` field to `Config`, `Env`, and all config lifecycle methods.

**Config struct** — add after `ConnTimeout`:

```go
TokenLifetime string `json:"token_lifetime"`
```

**Env struct** — add after `ConnTimeout`:

```go
TokenLifetime string
```

**`TokenLifetimeDuration` accessor** — add after `ConnTimeoutDuration`:

```go
func (c *Config) TokenLifetimeDuration() time.Duration {
	d, _ := time.ParseDuration(c.TokenLifetime)
	return d
}
```

**`loadDefaults`** — add at the end:

```go
if c.TokenLifetime == "" {
	c.TokenLifetime = "45m"
}
```

**`loadEnv`** — add at the end:

```go
if env.TokenLifetime != "" {
	if v := os.Getenv(env.TokenLifetime); v != "" {
		c.TokenLifetime = v
	}
}
```

**`Merge`** — add at the end:

```go
if overlay.TokenLifetime != "" {
	c.TokenLifetime = overlay.TokenLifetime
}
```

**`validate`** — add before the final `return nil`:

```go
if _, err := time.ParseDuration(c.TokenLifetime); err != nil {
	return fmt.Errorf("invalid token_lifetime: %w", err)
}
```

### Step 3: Update imports and add `NewWithCredential` to `pkg/database/database.go`

Update the import block. The blank import `_ "github.com/jackc/pgx/v5/stdlib"` becomes a named import since we now call `stdlib.OpenDB` directly (it still registers the `pgx` driver as a side effect). Add Azure SDK imports for token acquisition.

Replace the existing import block:

```go
import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"

	"github.com/JaimeStill/herald/pkg/lifecycle"
)
```

Add the `NewWithCredential` constructor after the existing `New` function:

```go
func NewWithCredential(cfg *Config, cred azcore.TokenCredential, logger *slog.Logger) (System, error) {
	connConfig, err := pgx.ParseConfig(cfg.Dsn())
	if err != nil {
		return nil, fmt.Errorf("parse database config: %w", err)
	}

	beforeConnect := stdlib.OptionBeforeConnect(
		func(ctx context.Context, cc *pgx.ConnConfig) error {
			token, err := cred.GetToken(ctx, policy.TokenRequestOptions{
				Scopes: []string{TokenScope},
			})
			if err != nil {
				return fmt.Errorf("acquire database token: %w", err)
			}

			cc.Password = token.Token
			return nil
		},
	)

	db := stdlib.OpenDB(*connConfig, beforeConnect)

	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.TokenLifetimeDuration())

	return &database{
		conn:        db,
		logger:      logger.With("system", "database"),
		connTimeout: cfg.ConnTimeoutDuration(),
	}, nil
}
```

## Validation Criteria

- [ ] `TokenLifetime` field on Config with 45m default, env override, merge, and validation
- [ ] `NewWithCredential` constructor compiles with correct signature
- [ ] `BeforeConnect` hook uses correct OSSRDBMS AAD scope
- [ ] `NewWithCredential` uses `cfg.TokenLifetimeDuration()` for `ConnMaxLifetime`
- [ ] `New` constructor unchanged — still uses `sql.Open("pgx", dsn)` with `ConnMaxLifetimeDuration()`
- [ ] Blank `_ "github.com/jackc/pgx/v5/stdlib"` import removed (now imported directly)
- [ ] `go vet ./...` passes
