# 106 — Add Token-Based Database Authentication

## Context

Issue #106 is part of Objective #97 (Managed Identity for Azure Services). It adds a `NewWithCredential` constructor to `pkg/database/` that uses pgx's `BeforeConnect` hook to acquire Entra tokens for PostgreSQL authentication, with forced 45-minute `ConnMaxLifetime` as a token expiry safety net.

## Files Modified

- `pkg/database/database.go` — Add `NewWithCredential` constructor

## Architecture Approach

Follow the same dual-constructor pattern established by `pkg/storage/storage.go` (#105). The existing `New` constructor is unchanged. `NewWithCredential` uses `pgx.ParseConfig()` → `stdlib.OpenDB()` with `stdlib.OptionBeforeConnect()` to inject a fresh Entra token as the connection password on every new connection.

Key details:
- `pgx.ParseConfig(dsn)` parses the DSN (password will be empty — that's fine)
- `stdlib.OptionBeforeConnect` injects a callback that calls `cred.GetToken()` with the OSSRDBMS AAD scope and sets `cc.Password = token.Token`
- `stdlib.OpenDB(connConfig, beforeConnectOpt)` returns `*sql.DB` — preserves the existing interface
- `ConnMaxLifetime` forced to 45 minutes regardless of config (Entra tokens expire ~1 hour)
- `validate()` already does not require password — no change needed

## Implementation

### Step 1: Add `NewWithCredential` to `pkg/database/database.go`

New imports needed:
- `github.com/Azure/azure-sdk-for-go/sdk/azcore`
- `github.com/Azure/azure-sdk-for-go/sdk/azcore/policy`
- `github.com/jackc/pgx/v5` (already indirect dep, now direct)
- `github.com/jackc/pgx/v5/stdlib` (already imported via blank import, now used directly)

Constructor signature mirrors storage:
```go
func NewWithCredential(cfg *Config, cred azcore.TokenCredential, logger *slog.Logger) (System, error)
```

Body:
1. `pgx.ParseConfig(cfg.Dsn())` — parse DSN into `*pgx.ConnConfig`
2. Build `OptionBeforeConnect` callback that acquires token via `cred.GetToken(ctx, policy.TokenRequestOptions{Scopes: []string{"https://ossrdbms-aad.database.windows.net/.default"}})`
3. `stdlib.OpenDB(*connConfig, beforeConnectOpt)` → `*sql.DB`
4. Configure pool: `SetMaxOpenConns`, `SetMaxIdleConns`, force `SetConnMaxLifetime(45 * time.Minute)`
5. Return `&database{conn, logger, connTimeout}`

The blank import `_ "github.com/jackc/pgx/v5/stdlib"` changes to a named import `"github.com/jackc/pgx/v5/stdlib"` since we now call `stdlib.OpenDB` directly. The blank import is no longer needed — `stdlib.OpenDB` registers the driver as a side effect of the package being imported.

## Validation Criteria

- [ ] `NewWithCredential` constructor compiles with correct signature
- [ ] `BeforeConnect` hook uses correct OSSRDBMS AAD scope
- [ ] `ConnMaxLifetime` forced to 45 minutes in credential mode
- [ ] `New` constructor unchanged — still uses `sql.Open("pgx", dsn)`
- [ ] Blank `_ "github.com/jackc/pgx/v5/stdlib"` import removed (now imported directly)
- [ ] `go vet ./...` passes
