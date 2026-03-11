# 119 — Add MSAL Auth Service with Login Gate

## Summary

Added the `Auth` service wrapping `@azure/msal-browser` v5.4.0 for Azure Entra ID authentication in the Lit SPA. The service reads server-injected config from the DOM, initializes MSAL, handles redirect login flow, and provides token acquisition. When auth is disabled (no config present), all methods are safe no-ops. The app bootstrap was converted to an async IIFE that gates on authentication before starting the router.

A configurable `CacheLocation` typed string enum was added to `pkg/auth` and threaded through `ClientAuthConfig` to the client, allowing operators to choose between `localStorage` (default) and `sessionStorage` for MSAL token caching via config or the `HERALD_AUTH_CACHE_LOCATION` env var.

## Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Cache location | Configurable via `CacheLocation` type | Operator flexibility — `localStorage` default, `sessionStorage` option via config/env var |
| Auth service location | `core/auth.ts` (not `domains/`) | Framework infrastructure, not a domain service |
| Bootstrap pattern | Async IIFE | Explicit bootstrap boundary, gates on auth before router start |
| MSAL version | v5.4.0 (resolved from `^5.4.0`) | Latest stable, API compatible with guide's v4 assumptions |
| Scope convention | `api://<client_id>/access_as_user` | Derived from config, no separate field needed |

## Files Modified

- `pkg/auth/config.go` — Added `CacheLocation` type, constants, field on Config/Env, wired into loadDefaults/loadEnv/Merge/validate
- `internal/config/config.go` — Added `HERALD_AUTH_CACHE_LOCATION` env var mapping
- `app/app.go` — Added `CacheLocation` field to `ClientAuthConfig`
- `cmd/server/modules.go` — Pass `CacheLocation` through to `ClientAuthConfig`
- `app/package.json` — Added `@azure/msal-browser` dependency
- `app/client/core/auth.ts` — New Auth service (PascalCase singleton)
- `app/client/core/index.ts` — Re-exported Auth
- `app/client/app.ts` — Async IIFE bootstrap with auth gate
- `tests/config/auth_test.go` — Added CacheLocation tests (defaults, validation, env override, merge)
- `tests/app/app_test.go` — Added cache_location to auth config injection test

## Patterns Established

- **Typed string enums for config values**: `CacheLocation` follows the `Mode` pattern — typed string, named constants, validated in `validate()`. Future config enums should follow this pattern.
- **TypeScript module-scoped `let` narrowing**: When using module-scoped `let` variables in object literal methods, TypeScript narrows based on initialization. Methods defined after the reassigning method see the full union type; explicit null guards needed otherwise.

## Validation Results

- `go test ./tests/...` — all 20 packages pass
- `go vet ./...` — clean
- `bun run build` — produces `dist/app.js` and `dist/app.css` without errors
