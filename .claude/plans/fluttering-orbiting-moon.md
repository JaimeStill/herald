# 113 — Auth Context Package and JWT Validation Middleware

## Context

Sub-issue 1 of Objective #98 (API Authentication Middleware). Establishes `pkg/auth/` as the unified auth package — moving existing auth config from `internal/config/`, adding JWT validation middleware infrastructure, and promoting `golang-jwt/jwt/v5` to direct dependency. Wiring auth middleware into the API module is #114's scope.

## Key Design Decision

**Unified auth config in `pkg/auth/`** — Move `internal/config/auth.go` → `pkg/auth/config.go`, add `Authority` field, switch to `Env` injection pattern. The middleware receives `*auth.Config` directly and checks `cfg.Mode == ModeAzure`. No separate middleware AuthConfig type, no mapping step.

This follows the established pattern: `database.Config` in `pkg/database/`, `storage.Config` in `pkg/storage/`.

## Files

| Action | File | Purpose |
|--------|------|---------|
| **Move** | `internal/config/auth.go` → `pkg/auth/config.go` | Auth config with Mode, ManagedIdentity, TenantID, ClientID, ClientSecret, Authority (new), TokenCredential(). Switch to Env injection pattern. |
| **Create** | `pkg/auth/errors.go` | Sentinel errors: `ErrUnauthorized`, `ErrTokenExpired`, `ErrInvalidToken` |
| **Create** | `pkg/auth/user.go` | `User` struct, `ContextWithUser`, `UserFromContext` |
| **Create** | `pkg/middleware/auth.go` | `Auth(cfg, logger)` middleware function |
| **Create** | `pkg/middleware/auth_jwks.go` | Unexported `jwksProvider` — OIDC discovery, JWKS cache, key rotation |
| **Update** | `internal/config/config.go` | `Auth auth.Config` (was `AuthConfig`), add `authEnv`, update finalize/merge |
| **Update** | `internal/infrastructure/infrastructure.go` | Import `auth.AgentScope` instead of `config.AgentScope` |
| **Update** | `tests/config/auth_test.go` | Import `pkg/auth` instead of `internal/config` for auth types |
| **Update** | `tests/infrastructure/infrastructure_test.go` | Same import update |
| **Update** | `tests/api/api_test.go` | Same import update |

## Implementation Steps

### Step 1: Create `pkg/auth/config.go` (move + extend)

Move `internal/config/auth.go` to `pkg/auth/config.go`. Changes:

- Package becomes `auth` (was `config`)
- Rename `AuthConfig` → `Config`, `AuthMode` → `Mode`, `AuthModeNone` → `ModeNone`, `AuthModeAzure` → `ModeAzure`
- Add `Authority string` field with `json:"authority"` tag
- Add `Env` struct (like `database.Env`):
  - Fields: Mode, ManagedIdentity, TenantID, ClientID, ClientSecret, Authority
- Change `Finalize()` → `Finalize(env *Env) error` with env injection
- Add `deriveDefaults()` call after `loadEnv` — sets Authority from TenantID if still empty: `https://login.microsoftonline.com/{tenant_id}/v2.0`
- Remove hardcoded `EnvAuth*` constants (env var names now injected)
- Keep `AgentScope`, `TokenCredential()`, `Merge()` as-is (with type renames)

### Step 2: Update `internal/config/config.go`

- Add import `"github.com/JaimeStill/herald/pkg/auth"`
- Add `var authEnv = &auth.Env{...}` with `HERALD_AUTH_*` env var names (+ new `HERALD_AUTH_AUTHORITY`)
- Change `Auth AuthConfig` → `Auth auth.Config`
- Change `c.Auth.Finalize()` → `c.Auth.Finalize(authEnv)`
- Change `c.Auth.Merge(&overlay.Auth)` — same call, new type

### Step 3: Update consumers

- `internal/infrastructure/infrastructure.go`: `config.AgentScope` → `auth.AgentScope`
- `tests/config/auth_test.go`: `config.AuthConfig` → `auth.Config`, `config.AuthModeNone` → `auth.ModeNone`, etc.
- `tests/infrastructure/infrastructure_test.go`: `config.AuthConfig{Mode: config.AuthModeNone}` → `auth.Config{Mode: auth.ModeNone}` (import update in test helper)
- `tests/api/api_test.go`: Same pattern

### Step 4: Delete `internal/config/auth.go`

Remove the original file after all references are updated.

### Step 5: Create `pkg/auth/errors.go`

Three sentinel errors:
- `ErrUnauthorized = errors.New("unauthorized")`
- `ErrTokenExpired = errors.New("token expired")`
- `ErrInvalidToken = errors.New("invalid token")`

### Step 6: Create `pkg/auth/user.go`

- Private context key: `type contextKey struct{}`, `var userKey = contextKey{}`
- `User` struct: `ID string` (from `oid`), `Name string` (from `name`/`preferred_username`), `Email string` (from `email`/`upn`)
- `ContextWithUser(ctx, *User) context.Context`
- `UserFromContext(ctx) *User` — returns nil if absent

### Step 7: Create `pkg/middleware/auth_jwks.go`

Unexported `jwksProvider` — all types/methods unexported, consumed only by `auth.go`:

- `newJWKSProvider(authority, logger)` — constructor with `http.Client` (10s timeout)
- `ensureDiscovered() error` — lazy OIDC discovery → extracts `jwks_uri` and `issuer`
- `getKey(kid) (*rsa.PublicKey, error)` — read-lock cache check, refresh on miss, double-check after write-lock
- `refresh() error` — fetch JWKS, parse RSA keys, replace map
- `issuer() string` — returns discovered issuer (read-locked)
- Thread safety: `sync.RWMutex` on keys/jwksURI/iss

### Step 8: Create `pkg/middleware/auth.go`

`Auth(cfg *auth.Config, logger *slog.Logger) func(http.Handler) http.Handler`:

- If `cfg.Mode != auth.ModeAzure` → pass-through
- Create `jwksProvider` from `cfg.Authority`
- HandlerFunc:
  1. Extract `Bearer <token>` from Authorization header
  2. `provider.ensureDiscovered()` — lazy first-request OIDC discovery
  3. `jwt.Parse(token, provider.keyFunc, WithValidMethods(["RS256"]), WithAudience(cfg.ClientID), WithIssuer(provider.issuer()), WithExpirationRequired())`
  4. On error → map to auth sentinel, `respondUnauthorized(w, err)`
  5. On success → extract claims → `auth.User` → `auth.ContextWithUser` → next

Inline `respondUnauthorized` helper (avoids coupling to `pkg/handlers`). Matches `{"error": "message"}` format.

Helpers: `claimString(claims, key)`, `firstNonEmpty(a, b)`.

### Step 9: Promote `golang-jwt/jwt/v5`

`go get github.com/golang-jwt/jwt/v5` then `go mod tidy`.

## Key Decisions

| Decision | Rationale |
|----------|-----------|
| Unified config in `pkg/auth/` | Follows database/storage pattern. One config type, no mapping. |
| Middleware checks `Mode == ModeAzure` | No separate `Enabled` bool — derive from existing config field |
| Discovered issuer from OIDC config | Authoritative across commercial/gov clouds |
| `ensureDiscovered()` before `jwt.Parse` | Issuer must be known before parser options are set |
| `sync.RWMutex` + double-check in refresh | Prevents thundering herd on key rotation |
| `deriveDefaults()` after loadEnv | Authority depends on TenantID which may come from env vars |
| RSA-only key parsing | Azure Entra ID uses RS256 exclusively |
| Inline 401 response in middleware | Avoids coupling `pkg/middleware` to `pkg/handlers` |

## Validation

- [ ] `go vet ./...` passes
- [ ] `go build ./...` passes
- [ ] `go test ./tests/...` passes (existing tests updated for import changes)
- [ ] `pkg/auth/` exports Config, Mode constants, User, context helpers, error sentinels, AgentScope, TokenCredential
- [ ] `pkg/middleware/auth.go` compiles with correct function signature
- [ ] JWKS provider handles OIDC discovery and key caching
- [ ] `Mode != ModeAzure` produces a pass-through middleware
- [ ] `golang-jwt/jwt/v5` is a direct dependency in go.mod
- [ ] `internal/config/auth.go` is deleted
