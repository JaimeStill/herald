# Objective: API Authentication Middleware

**Issue:** [#98](https://github.com/JaimeStill/herald/issues/98)
**Phase:** Phase 4 — Security and Deployment (v0.4.0)

## Scope

Add JWT bearer token validation middleware for Azure Entra ID authentication on the HTTP API. Extract user identity into request context for audit trail use. All auth features are opt-in — `auth_mode: "none"` (default) skips middleware entirely for zero-friction local development.

### What's Covered

- JWT validation middleware: extract Bearer token, validate against Entra JWKS, verify audience/issuer/expiry
- `pkg/auth/` package with `User` type and context helpers (`UserFromContext`, `ContextWithUser`)
- Auth middleware config with OIDC discovery for JWKS fetching
- Wire auth middleware into API module between CORS and Logger
- Classifications domain: populate `validated_by` from authenticated user identity
- Health/readiness endpoints remain unauthenticated (root router, not API module)

### What's Not Covered

- Web client MSAL.js integration (Objective 5, #99)
- Managed identity for services (Objective 3, #97 — complete)

## Sub-Issues

| # | Title | Issue | Status | Dependencies |
|---|-------|-------|--------|--------------|
| 1 | Add auth context package and JWT validation middleware | [#113](https://github.com/JaimeStill/herald/issues/113) | Open | None |
| 2 | Wire auth middleware into API module and config pipeline | [#114](https://github.com/JaimeStill/herald/issues/114) | Open | #113 |
| 3 | Populate validated_by from authenticated user identity | [#115](https://github.com/JaimeStill/herald/issues/115) | Open | #113, #114 |

Sub-issues are strictly sequential: 1 → 2 → 3.

## Architecture Decisions

- **Derive "enabled" from `AuthConfig.Mode`** — `Mode == "azure"` means enforce JWT validation. No separate `enabled` bool on the internal config. The middleware's own `AuthConfig.Enabled` is set at wiring time in `api.NewModule()`.
- **Separate middleware AuthConfig** — `pkg/middleware/AuthConfig` is distinct from `internal/config/AuthConfig`. The internal config holds Azure credential params; the middleware config holds JWT validation params. Mapping happens at wiring time.
- **OIDC discovery for JWKS** — Fetch `{authority}/.well-known/openid-configuration` → extract `jwks_uri` → fetch and cache public signing keys. Refresh on unknown `kid` (key rotation). More robust than hardcoding JWKS URL patterns across cloud environments (commercial vs gov).
- **Authority URL for gov cloud** — Default authority is `https://login.microsoftonline.com/{tenant_id}/v2.0`. Gov cloud overrides via `HERALD_API_AUTH_AUTHORITY`.
- **`golang-jwt/jwt/v5` promoted to direct** — Already indirect via `azidentity`. No new dependency tree.
- **Display name for validated_by** — Use the `name` JWT claim for human-readable audit trail, fallback to `preferred_username`.
