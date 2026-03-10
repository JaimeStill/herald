# 106 — Add Token-Based Database Authentication

## Summary

Added `NewWithCredential` constructor to `pkg/database/` that authenticates to Azure PostgreSQL using Entra tokens via a pgx `BeforeConnect` hook. The hook acquires a fresh token on each new connection using the `azcore.TokenCredential` interface. A configurable `TokenLifetime` field (default 45m) controls `ConnMaxLifetime` to force connection recycling before Entra token expiry (~1 hour).

## Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Token scope | Package-level `TokenScope` constant | Same across all Azure cloud instances; provides a single reference point |
| Token lifetime | Configurable `TokenLifetime` on Config (default 45m) | Allows tuning per environment while preserving the safety net against token expiry |
| Constructor pattern | `NewWithCredential` alongside `New` | Mirrors `pkg/storage/` dual-constructor pattern from #105 |
| Hook mechanism | `stdlib.OptionBeforeConnect` | pgx v5 stdlib provides this option for `OpenDB`, avoids switching to `pgxpool` |

## Files Modified

- `pkg/database/constants.go` (new) — `TokenScope` constant
- `pkg/database/config.go` — `TokenLifetime` field, `TokenLifetimeDuration()` accessor, defaults/env/merge/validate
- `pkg/database/database.go` — `NewWithCredential` constructor, updated imports (blank → named stdlib, added azcore/pgx)
- `internal/config/config.go` — `HERALD_DB_TOKEN_LIFETIME` env var mapping

## Patterns Established

- Database token scope as a package-level constant (`TokenScope`) for reuse by infrastructure wiring
- Configurable token lifetime separate from connection max lifetime — each auth mode controls its own pool recycling

## Validation Results

- `go vet ./...` passes clean
- No integration tests (requires Azure credentials + PostgreSQL — per project convention)
