# 114 + 115 — Wire Auth Middleware and Populate validated_by

## Summary

Wired the auth middleware into the API module and connected authenticated user identity to the classifications domain. Combined #114 (middleware wiring) and #115 (validated_by population) into a single session since #114 was a one-line change. Together they complete Objective #98 (API Authentication Middleware).

## Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| No `APIConfig.Auth` field | Pass `Config.Auth` directly to middleware | #113 implemented middleware to accept `*auth.Config` — no mapping layer needed |
| No `config.json` changes | Top-level `auth` section already sufficient | Adding `api.auth` would duplicate config without purpose |
| Conditional override pattern | `if user := auth.UserFromContext(ctx)` | Preserves backward compatibility when auth is disabled |

## Files Modified

- `internal/api/api.go` — Added `middleware.Auth` registration between CORS and Logger
- `internal/classifications/handler.go` — Added `pkg/auth` import, `UserFromContext` override in Validate and Update handlers
- `tests/middleware/middleware_test.go` — Added 4 auth middleware tests (ModeNone passthrough, no token required, missing bearer, invalid bearer)
- `tests/classifications/handler_test.go` — Added 4 handler auth context tests (validate/update with and without authenticated user)

## Patterns Established

- **Auth context override pattern**: `if user := auth.UserFromContext(r.Context()); user != nil { cmd.Field = user.Name }` — use in any handler that needs to derive identity from JWT when available, falling back to request body when auth is disabled.

## Validation Results

- `go vet ./...` — clean
- `go build ./cmd/server/` — success
- 20/20 test packages pass (8 new tests added across middleware and classifications)
