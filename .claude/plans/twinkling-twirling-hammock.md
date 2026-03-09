# Plan: #103 — Add AuthConfig and credential provider infrastructure

## Context

Objective #96 (Azure Identity Credential Infrastructure) requires adding the Azure Identity SDK and creating a credential provider foundation for managed identity support. This is the first task in Phase 4's security track — downstream objectives (#97 Managed Identity, #98 Auth Middleware) depend on the `Credential` field added here.

## Approach

Add `AuthConfig` to `internal/config/` following the `ServerConfig` same-package pattern (direct env constants, `Finalize()`/`Merge()`). Wire a `TokenCredential()` factory method on `AuthConfig` that returns `nil` for `none` mode and `DefaultAzureCredential` for `azure` mode. Add the credential to `Infrastructure` and propagate through `api.NewRuntime()`.

## Files to Modify

| File | Change |
|------|--------|
| `internal/config/auth.go` | **New** — `AuthConfig` struct with `Finalize()`, `Merge()`, `TokenCredential()` |
| `internal/config/config.go` | Add `Auth AuthConfig` field, wire finalize/merge |
| `internal/infrastructure/infrastructure.go` | Add `Credential azcore.TokenCredential` field, create in `New()` |
| `internal/api/runtime.go` | Propagate `Credential` in `NewRuntime()` |
| `go.mod` / `go.sum` | Add `azidentity` dependency |

## Implementation Steps

### Step 1: Add `azidentity` dependency

```bash
go get github.com/Azure/azure-sdk-for-go/sdk/azidentity
```

### Step 2: Create `internal/config/auth.go`

New file with:
- Env constants: `EnvAuthMode`, `EnvAuthTenantID`, `EnvAuthClientID`, `EnvAuthClientSecret`
- `AuthConfig` struct: `Mode`, `TenantID`, `ClientID`, `ClientSecret` (all string, json-tagged)
- `Finalize()` → `loadDefaults()` (mode=`none`) → `loadEnv()` → `validate()` (mode must be `none` or `azure`)
- `Merge()` — non-zero field overwrite
- `TokenCredential()` factory → returns `(azcore.TokenCredential, error)`: nil for `none`, `DefaultAzureCredential` for `azure` with options (TenantID, ClientID, ClientSecret populate `ClientSecretCredential` options when all three are set, otherwise bare `DefaultAzureCredential`)

**Design note on credential selection**: When `TenantID`, `ClientID`, and `ClientSecret` are all provided, use `azidentity.NewClientSecretCredential()` (explicit service principal). Otherwise, use `azidentity.NewDefaultAzureCredential()` which walks the Azure credential chain (managed identity, workload identity, Azure CLI, etc.).

### Step 3: Wire `AuthConfig` into root `Config`

In `config.go`:
- Add `Auth AuthConfig` field with `json:"auth"` tag
- Add `c.Auth.Merge(&overlay.Auth)` in `Merge()`
- Add `c.Auth.Finalize()` call in `finalize()` (before agent, since downstream objectives will need credential for agent config)

### Step 4: Add `Credential` to `Infrastructure`

In `infrastructure.go`:
- Add `Credential azcore.TokenCredential` field (import `azcore`)
- In `New()`, call `cfg.Auth.TokenCredential()` and assign result
- Credential is nil when mode is `none` — all existing behavior unchanged

### Step 5: Propagate through `api.NewRuntime()`

In `runtime.go`:
- Add `Credential: infra.Credential` to the `Infrastructure` literal in `NewRuntime()`

## Validation

- `go build ./...` succeeds
- `go vet ./...` passes
- `go test ./tests/...` passes
- Default config (no auth section): `Credential` is nil, all existing behavior unchanged
- Config with `"auth": {"mode": "azure"}`: `TokenCredential()` returns a valid credential
- Config with invalid mode (e.g., `"auth": {"mode": "foo"}`): validation error
- `HERALD_AUTH_*` env vars override JSON config values
