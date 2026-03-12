# 124 - Make AgentScope Configurable for Azure Government

## Summary

Converted hardcoded `AgentScope` (auth) and `TokenScope` (database) constants to configurable fields on their respective Config structs. Both use commercial Azure defaults and can be overridden via `HERALD_AUTH_AGENT_SCOPE` and `HERALD_DB_TOKEN_SCOPE` env vars for Gov cloud deployments.

## Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Include TokenScope | Configurable alongside AgentScope | Provides config-level escape hatch if scope ever changes, consistent pattern across both scopes |
| No validation on scope values | Accept any string | Scope URLs vary by cloud and could change; validation would be fragile |
| Parameter injection for infrastructure | Pass scope string into `newAgentFactory` | Removes infrastructure's dependency on `pkg/auth` import |

## Files Modified

- `pkg/auth/config.go` — Removed `AgentScope` constant, added `AgentScope` field to Config and Env, wired into loadDefaults/loadEnv/Merge
- `pkg/database/config.go` — Added `TokenScope` field to Config and Env, wired into loadDefaults/loadEnv/Merge
- `pkg/database/database.go` — Removed `TokenScope` constant, read `cfg.TokenScope` in NewWithCredential
- `internal/config/config.go` — Added env var mappings for both scopes
- `internal/infrastructure/infrastructure.go` — Pass `cfg.Auth.AgentScope` into `newAgentFactory`, removed `pkg/auth` import
- `tests/config/auth_test.go` — Added AgentScope default, env override, merge, and explicit tests
- `tests/database/config_test.go` — Added TokenScope default, env override, merge, merge-preserve, and explicit tests

## Patterns Established

- OAuth scope constants that vary by cloud environment should be configurable Config fields with commercial defaults, not package-level constants

## Validation Results

- `go vet ./...` — pass
- `go build ./cmd/server/` — pass
- `go test ./tests/...` — 20/20 packages pass
- No references to `auth.AgentScope` or `database.TokenScope` constants in source code
