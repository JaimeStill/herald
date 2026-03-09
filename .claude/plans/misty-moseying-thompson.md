# Objective Planning: #96 — Azure Identity Credential Infrastructure

## Context

Phase 4 (Security and Deployment) requires Azure managed identity support for connecting to Azure services without connection strings. Objective #96 lays the foundation by adding the `azidentity` SDK dependency, creating an `AuthConfig` with the three-phase finalize pattern, and wiring credential creation into `infrastructure.New()`. The credential is optional — nil when `auth_mode` is `connection_string` (the default), preserving current behavior. Objective 3 will later wire the credential into storage, database, and agent clients.

## Transition Closeout

**Objective #95** (Docker Production Image): 1/1 sub-issues complete (100%). Clean transition.

- Close issue #95
- Update `_project/phase.md` — mark Objective 1 as Complete, Objective 2 as Active
- Delete `_project/objective.md`

## Sub-Issue Decomposition

**Single sub-issue.** The scope is tightly coupled (config drives credential, credential slots into infrastructure) and small (~140 lines of new code). Splitting would create artificial PRs with dead code in the first and an incomprehensible second.

### Sub-Issue #1: Add AuthConfig and credential provider infrastructure

**Labels:** `infrastructure`, `config`
**Milestone:** v0.4.0 - Security and Deployment

#### Scope

1. **Add `azidentity` dependency** — `go get github.com/Azure/azure-sdk-for-go/sdk/azidentity`

2. **Create `internal/config/auth.go`** — follows `ServerConfig` pattern (same-package, direct env constants)
   - Struct: `AuthMode` (string), `TenantID`, `ClientID`, `ClientSecret`
   - Env constants: `HERALD_AUTH_MODE`, `HERALD_AUTH_TENANT_ID`, `HERALD_AUTH_CLIENT_ID`, `HERALD_AUTH_CLIENT_SECRET`
   - Default: `AuthMode = "none"`
   - Validation: mode must be `none` or `azure`; when `azure`, `ClientID` required
   - `Finalize()`, `Merge()`, `loadDefaults()`, `loadEnv()`, `validate()`

3. **Wire into root `Config`** (`internal/config/config.go`)
   - Add `Auth AuthConfig` field with `json:"auth"` tag
   - Add `c.Auth.Finalize()` in `finalize()`
   - Add `c.Auth.Merge(&overlay.Auth)` in `Merge()`

4. **Create credential factory** — `AuthConfig.TokenCredential() (azcore.TokenCredential, error)`
   - Method on `AuthConfig` (not a separate package — the factory is trivial and config owns the mode decision)
   - Returns `nil, nil` for `none` mode
   - Returns `azidentity.DefaultAzureCredential` for `azure` mode, configured with `ClientID` and `TenantID`

5. **Wire into `Infrastructure`** (`internal/infrastructure/infrastructure.go`)
   - Add `Credential azcore.TokenCredential` field
   - Call `cfg.Auth.TokenCredential()` in `New()`, store result (nil for connection_string)
   - No lifecycle hooks needed

6. **Propagate through `api.NewRuntime`** (`internal/api/runtime.go`)
   - Copy `Credential` into module-scoped Infrastructure copy

#### Acceptance Criteria

- [ ] `go build ./...` succeeds with `azidentity` dependency
- [ ] `none` mode (default): `Infrastructure.Credential` is nil, all existing behavior unchanged
- [ ] `azure` mode: `Infrastructure.Credential` is a valid `azcore.TokenCredential`
- [ ] `HERALD_AUTH_*` env vars override JSON config values
- [ ] Invalid `auth_mode` values produce a clear validation error
- [ ] `go vet ./...` and `go test ./tests/...` pass

#### Key Files

| File | Change |
|------|--------|
| `internal/config/auth.go` | **New** — AuthConfig struct with finalize pattern + TokenCredential factory |
| `internal/config/config.go` | Add Auth field, Merge, finalize wiring |
| `internal/infrastructure/infrastructure.go` | Add Credential field, create in New() |
| `internal/api/runtime.go` | Propagate Credential through module-scoped copy |
| `go.mod` / `go.sum` | Add azidentity dependency |

#### Design Decisions

- **Factory on config, not a separate package**: `TokenCredential()` is a method on `AuthConfig` rather than a `pkg/credential/` package. The factory is ~10 lines and config owns the mode decision. If Objective 3 needs credential access, it gets it from `Infrastructure.Credential` (already exposed).
- **`DefaultAzureCredential`**: Wraps the full Azure credential chain (managed identity, workload identity, Azure CLI, etc.) rather than specific credential types. This handles all deployment scenarios automatically.
- **`TenantID` optional**: `DefaultAzureCredential` can discover tenant from the environment. Only `ClientID` is required for `azure` mode to target the correct user-assigned identity.
- **No `client_secret` validation**: Secret is optional — supports both user-assigned managed identity (no secret) and service principal (with secret) via the same config.
- **Auth mode naming**: `none`/`azure` rather than `connection_string`/`managed_identity`. Describes the identity provider, not the connection mechanism. More intuitive and extensible.

## Verification

1. `go build ./...` — compiles with new dependency and code
2. `go vet ./...` — no issues
3. `go test ./tests/...` — existing tests pass
4. Manual: run `mise run dev` with no `auth` config → mode defaults to `none`, credential is nil, service works as before
5. Manual: set `HERALD_AUTH_MODE=invalid` → startup fails with validation error
