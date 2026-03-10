# Objective Planning: #98 — API Authentication Middleware

## Transition Closeout (Objective #97)

Objective #97 (Managed Identity for Azure Services) is **100% complete** — all 4 sub-issues closed. No incomplete work to disposition.

**Actions:**
- Close issue #97
- Update `_project/phase.md` — mark Objective 3 as Complete, Objective 4 as Active
- Delete `_project/objective.md` and replace with #98 content

---

## Context

Phase 4 focuses on making Herald deployable to IL4/IL6 Azure Government. Objective #98 adds JWT bearer token validation to the HTTP API so that requests are authenticated via Azure Entra ID. This is the user-facing authentication layer — Objective #96 (credential infrastructure) and #97 (managed identity for services) handle service-to-service auth. Objective #99 (MSAL.js web client) depends on this one.

The existing `AuthConfig` already has `Mode` ("none"/"azure"), `TenantID`, and `ClientID`. The middleware system uses a stack-based composition pattern (`Use`/`Apply`). `golang-jwt/jwt/v5` is already an indirect dependency via `azidentity`.

## Sub-Issues (3)

### Sub-Issue 1: Add auth context package and JWT validation middleware

**Scope:** Foundation layer — pure `pkg/` work, no `internal/` changes.

**Deliverables:**

1. **`pkg/auth/`** — New package with:
   - `user.go` — `User` struct (ID, Name, Email from JWT claims `oid`, `name`/`preferred_username`, `email`/`upn`), `UserFromContext(ctx) (*User, bool)`, `ContextWithUser(ctx, *User) context.Context`
   - `errors.go` — `ErrUnauthorized`, `ErrTokenExpired`, `ErrInvalidToken`

2. **`pkg/middleware/auth.go`** — `Auth(cfg *AuthConfig, logger *slog.Logger) func(http.Handler) http.Handler`:
   - Extract `Bearer` token from `Authorization` header
   - Validate JWT signature via JWKS, verify `aud` (ClientID), `iss` (tenant), `exp`
   - On success: extract claims → `auth.User` → inject into context
   - On failure: 401 JSON error response
   - When `Enabled == false`: pass-through no-op

3. **`pkg/middleware/auth_config.go`** — `AuthConfig` struct:
   - `Enabled bool`, `TenantID string`, `ClientID string` (audience), `Authority string` (OIDC authority URL — defaults to `https://login.microsoftonline.com/{tenant_id}/v2.0`, overridable for gov cloud)
   - `Finalize(env *AuthEnv)` + `Merge(overlay)` following `CORSConfig` pattern
   - JWKS URL derived from Authority

4. **JWKS handling** — OIDC discovery flow: fetch `{authority}/.well-known/openid-configuration` to get `jwks_uri`, then fetch and cache public signing keys. Refresh on key-miss (unknown `kid` in incoming JWT) or periodic timer. Promote `golang-jwt/jwt/v5` from indirect to direct dependency.

**Labels:** `middleware`, `auth`
**Dependencies:** None

### Sub-Issue 2: Wire auth middleware into API module and config pipeline

**Scope:** Integration layer — connects library code to the application.

**Deliverables:**

1. **`internal/config/api.go`** — Add `Auth middleware.AuthConfig` field to `APIConfig` with env var wiring (`HERALD_API_AUTH_ENABLED`, `HERALD_API_AUTH_AUTHORITY`)
   - `TenantID` and `ClientID` derived from existing `AuthConfig` fields at wiring time (no duplication)

2. **`internal/api/api.go`** — Register auth middleware between CORS and Logger:
   ```
   m.Use(middleware.CORS(&cfg.API.CORS))
   m.Use(middleware.Auth(authCfg, logger))  // NEW
   m.Use(middleware.Logger(logger))
   ```
   - Build `middleware.AuthConfig` by mapping: `Enabled` from `cfg.Auth.Mode == AuthModeAzure`, `TenantID`/`ClientID` from `cfg.Auth`

3. **`config.json`** — Add `api.auth` section with defaults (`enabled: false`)

**Labels:** `config`, `api`
**Dependencies:** Sub-Issue 1

### Sub-Issue 3: Populate validated_by from authenticated user identity

**Scope:** Domain layer — classifications handlers extract identity from JWT context.

**Deliverables:**

1. **`internal/classifications/handler.go`** — Update `Validate` and `Update` handlers:
   - When `auth.UserFromContext(ctx)` returns a user: override `cmd.ValidatedBy`/`cmd.UpdatedBy` with authenticated identity
   - When no user in context (auth disabled): preserve existing behavior (use request body value)
   - Identity format: `user.Name` (display name from `name` claim — human-readable audit trail), fallback to `preferred_username` if name empty

2. No schema changes — `validated_by` is already `TEXT`

**Labels:** `classifications`
**Dependencies:** Sub-Issues 1, 2

## Dependency Graph

```
Sub-Issue 1 (pkg/auth + pkg/middleware/auth)
    │
    ▼
Sub-Issue 2 (wire into API module + config)
    │
    ▼
Sub-Issue 3 (classifications identity)
```

## Architecture Decisions

- **Derive "enabled" from `AuthConfig.Mode`** — `Mode == "azure"` means enforce JWT validation. No separate `enabled` bool on the internal config. The middleware's own `AuthConfig.Enabled` is set at wiring time.
- **Separate middleware AuthConfig** — `pkg/middleware/AuthConfig` is distinct from `internal/config/AuthConfig`. The internal config holds Azure credential params; the middleware config holds JWT validation params. Mapping happens in `api.NewModule()`.
- **Authority URL for gov cloud** — Default authority is `https://login.microsoftonline.com/{tenant_id}/v2.0`. Gov cloud overrides via `HERALD_API_AUTH_AUTHORITY` (e.g., `https://login.microsoftonline.us/{tenant_id}/v2.0`).
- **`golang-jwt/jwt/v5` promoted to direct** — Already indirect via `azidentity`. No new dependency tree.
- **No `go-oidc`** — JWKS fetch + JWT validation with `golang-jwt` is straightforward and avoids an additional dependency.
- **OIDC discovery for JWKS** — Fetch `{authority}/.well-known/openid-configuration` → extract `jwks_uri` → fetch public keys. Refresh strategy: cache keys, refresh on unknown `kid` (key rotation) or periodic timer. More robust than hardcoding JWKS URL patterns across cloud environments.
- **Display name for validated_by** — Use the `name` JWT claim for human-readable audit trail, fallback to `preferred_username`.

## Verification

- `go vet ./...` passes
- `go test ./tests/...` passes
- With `auth_mode: "none"`: all endpoints accessible without tokens (existing behavior)
- With `auth_mode: "azure"` + valid tenant/client: unauthenticated requests get 401, valid Bearer tokens pass through, `validated_by` populated from token claims
- Health/readiness endpoints always accessible (root router, not API module)
