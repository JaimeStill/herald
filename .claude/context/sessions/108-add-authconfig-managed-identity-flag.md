# 108 - Add AuthConfig managed_identity flag and infrastructure wiring

## Summary

Added `ManagedIdentity` boolean to `AuthConfig` with `HERALD_AUTH_MANAGED_IDENTITY` env var support and wired credential-based constructors into `infrastructure.New()`. When `auth_mode: "azure"` and `managed_identity: true`, infrastructure uses `database.NewWithCredential()`, `storage.NewWithCredential()`, and a bearer token agent factory that acquires fresh Entra tokens per agent creation. The `auth_mode: "none"` path and `managed_identity: false` path are completely unchanged.

## Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Infrastructure structure | Extract `initSystems`, `initManagedSystems`, `newAgentFactory` private helpers | Keeps `New()` clean; each auth path is self-contained |
| Options map cloning | `maps.Clone(pc.Options)` | Avoids concurrent map writes from errgroup goroutines; cleaner than manual iteration |
| Scope constant locations | `AgentScope` in `auth.go`, `TokenScope` in `database.go` | Auth/domain-scoped constants belong with their related code, not standalone constants files |
| Managed identity env values | `"true"` or `"1"` only | Explicit truthy values; all other strings treated as false |

## Files Modified

- `internal/config/auth.go` — `ManagedIdentity` field, `EnvAuthManagedIdentity`, `AgentScope` constant, Merge/loadEnv support
- `internal/infrastructure/infrastructure.go` — `initSystems`, `initManagedSystems`, `newAgentFactory` helpers; simplified `New()`
- `pkg/database/database.go` — `TokenScope` constant moved here from `constants.go`
- `tests/config/auth_test.go` — tests for ManagedIdentity defaults, env overrides (table-driven), merge, JSON overlay

## Files Removed

- `internal/config/constants.go` — `AgentScope` moved to `auth.go`
- `pkg/database/constants.go` — `TokenScope` moved to `database.go`

## Patterns Established

- **Infrastructure init helpers**: Private functions encapsulate each auth path (`initSystems` dispatches, `initManagedSystems` handles credential constructors, `newAgentFactory` builds auth-mode-aware closure). New auth-dependent infrastructure follows the same extraction pattern.
- **Scope constant colocation**: OAuth scope constants live alongside the code that uses them, not in separate constants files.

## Validation Results

- `go vet ./...` — passes
- `go build ./cmd/server/` — passes
- `go test ./tests/...` — all 20 packages pass
- Local development workflow verified (classification still works with `auth_mode: "none"`)
