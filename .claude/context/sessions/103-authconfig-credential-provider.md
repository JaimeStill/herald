# 103 - Add AuthConfig and credential provider infrastructure

## Summary

Added Azure Identity SDK dependency and created `AuthConfig` with the three-phase finalize pattern, wiring credential creation into `infrastructure.New()` and propagating through `api.NewRuntime()`. The credential is optional — nil when `auth_mode` is `none` (default), preserving all existing behavior. When `auth_mode` is `azure`, the factory selects `ClientSecretCredential` (explicit service principal) or `DefaultAzureCredential` (full credential chain) based on whether all three SP fields are provided.

## Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Auth mode type | `AuthMode` typed string enum | Compile-time safety, self-documenting valid values |
| Credential selection | `ClientSecretCredential` when all SP fields set, else `DefaultAzureCredential` | Covers both explicit SP config and managed identity/CLI scenarios |
| Config location | `internal/config/auth.go` (same-package) | Follows `ServerConfig` pattern — direct env constants, no sub-package needed |
| Factory placement | Method on `AuthConfig` | Factory is trivial, config owns the mode decision |

## Files Modified

- `internal/config/auth.go` — **New**: `AuthMode` type, `AuthConfig` struct with `Finalize`/`Merge`/`TokenCredential`
- `internal/config/config.go` — Added `Auth AuthConfig` field, wired finalize/merge
- `internal/infrastructure/infrastructure.go` — Added `Credential azcore.TokenCredential` field, created in `New()`
- `internal/api/runtime.go` — Propagated `Credential` in `NewRuntime()`
- `config.json` — Added explicit `auth.auth_mode: "none"` section
- `go.mod` / `go.sum` — Added `azidentity` dependency
- `tests/config/auth_test.go` — **New**: 10 tests for AuthConfig
- `tests/infrastructure/infrastructure_test.go` — Added `Auth` field to `validConfig()`
- `tests/api/api_test.go` — Added `Auth` field to `validConfig()`

## Patterns Established

- Typed string enum for config mode fields (`AuthMode` with `AuthModeNone`/`AuthModeAzure`)
- Credential as an optional `Infrastructure` field (nil-safe for disabled modes)

## Validation Results

- `go build ./...` — pass
- `go vet ./...` — pass
- `go test ./tests/...` — 20/20 packages pass (10 new auth config tests)
