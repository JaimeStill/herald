# 113 — Add Auth Context Package and JWT Validation Middleware

## Summary

Established `pkg/auth/` as the unified authentication package and added JWT validation middleware to `pkg/middleware/`. Moved auth config from `internal/config/auth.go` to `pkg/auth/config.go` following the package-owns-config pattern (consistent with `database.Config`, `storage.Config`). Added `go-oidc` (coreos) for OIDC discovery and token verification instead of hand-rolling JWKS infrastructure. Documented dependency evaluation criteria in the project README.

## Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Auth config location | Unified in `pkg/auth/` | Follows established pattern where packages own their config types |
| No separate middleware config | Middleware receives `*auth.Config` directly | Eliminates mapping step; middleware checks `Mode == ModeAzure` |
| `go-oidc` for token verification | Replace ~150 lines of hand-rolled JWKS code | Battle-tested (Kubernetes uses it), eliminates security-sensitive RSA key decoding and cache management |
| Lazy OIDC provider init | `sync.Once` on first request | Avoids blocking server startup with a network call |
| Authority as config field | Configurable with derived default | Supports Azure Government cloud which uses different authority URLs |
| Dependency criteria codified | Added to `_project/README.md` | Sets precedent: purpose-built, authoritative, maintained, production-ready, not a supply chain risk |

## Files Modified

- `pkg/auth/config.go` — New: unified auth config (moved from `internal/config/auth.go` + Authority field + Env injection pattern)
- `pkg/auth/errors.go` — New: ErrUnauthorized, ErrTokenExpired, ErrInvalidToken sentinels
- `pkg/auth/user.go` — New: User struct, ContextWithUser, UserFromContext
- `pkg/middleware/auth.go` — New: Auth middleware with go-oidc verification
- `internal/config/auth.go` — Deleted (moved to `pkg/auth/config.go`)
- `internal/config/config.go` — Updated: `Auth auth.Config`, authEnv wiring, Finalize(authEnv)
- `internal/infrastructure/infrastructure.go` — Updated: `auth.AgentScope` import
- `_project/README.md` — Updated: go-oidc dependency listing, dependency criteria section
- `CHANGELOG.md` — Updated: v0.4.0-dev.98.113 entry
- `go.mod` / `go.sum` — Updated: go-oidc dependency
- `tests/config/auth_test.go` — Updated: auth package imports, Env injection for env override tests
- `tests/infrastructure/infrastructure_test.go` — Updated: auth package imports
- `tests/api/api_test.go` — Updated: auth package imports

## Patterns Established

- **Package-owns-config**: Auth config lives in `pkg/auth/` alongside the types it configures, not in `internal/config/`
- **Dependency criteria**: External libraries must be purpose-built, authoritative, maintained, production-ready, and not a supply chain risk
- **`deriveDefaults()` phase**: For config fields whose defaults depend on values from env var overrides, add a derivation step between loadEnv and validate
