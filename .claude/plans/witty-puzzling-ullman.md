# 108 - Add AuthConfig managed_identity flag and infrastructure wiring

## Context

Issue #108 is the integration sub-issue of Objective #97 (Managed Identity for Azure Services). The three dependency sub-issues are merged:
- #105 added `storage.NewWithCredential()`
- #106 added `database.NewWithCredential()`
- #107 added the `NewAgent` factory closure pattern on `Infrastructure`

This task adds the `ManagedIdentity` boolean to `AuthConfig` and branches `infrastructure.New()` to use credential-based constructors when the flag is true. The agent factory closure (currently always using the API key path) gets a bearer token branch that clones the provider config and injects a fresh Entra token per agent creation.

## Implementation

### Step 1: Add `ManagedIdentity` to `AuthConfig`

**File:** `internal/config/auth.go`

- Add `EnvAuthManagedIdentity = "HERALD_AUTH_MANAGED_IDENTITY"` constant
- Add `ManagedIdentity bool` field to `AuthConfig` (json: `"managed_identity"`)
- Update `Merge()` to overlay `ManagedIdentity` (bool — overlay `true` overwrites)
- Update `loadEnv()` to read `HERALD_AUTH_MANAGED_IDENTITY` (truthy: `"true"`, `"1"`)
- No `loadDefaults()` change needed (zero-value `false` is correct default)
- No `validate()` change needed (flag is valid in any mode; ignored when mode is `"none"`)

### Step 2: Add `AgentScope` constant

**File:** `internal/config/auth.go`

- Add `AgentScope = "https://cognitiveservices.azure.com/.default"` — the OAuth scope for AI Foundry bearer tokens, used by the agent factory closure in infrastructure wiring.

### Step 3: Branch `infrastructure.New()` on `ManagedIdentity`

**File:** `internal/infrastructure/infrastructure.go`

When `cred != nil && cfg.Auth.ManagedIdentity`:
- Call `database.NewWithCredential(&cfg.Database, cred, logger)` instead of `database.New()`
- Call `storage.NewWithCredential(&cfg.Storage, cred, logger)` instead of `storage.New()`
- Build bearer token agent factory closure:
  ```go
  newAgent := func(ctx context.Context) (agent.Agent, error) {
      tok, err := cred.GetToken(ctx, policy.TokenRequestOptions{
          Scopes: []string{config.AgentScope},
      })
      if err != nil {
          return agent.Agent{}, fmt.Errorf("acquire agent token: %w", err)
      }
      // Clone provider config to avoid concurrent map writes
      pc := agentCfg.Provider
      opts := make(map[string]string, len(pc.Options)+2)
      for k, v := range pc.Options {
          opts[k] = v
      }
      opts["token"] = tok.Token
      opts["auth_type"] = "bearer"
      pc.Options = opts
      cloned := agentCfg
      cloned.Provider = pc
      return agent.New(&cloned)
  }
  ```

When `cred == nil || !cfg.Auth.ManagedIdentity`:
- Existing path: `database.New()`, `storage.New()`, API key agent factory

### Step 4: Add `azcore/policy` import

**File:** `internal/infrastructure/infrastructure.go`

- Add `"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"` for `TokenRequestOptions`

## Files Modified

| File | Change |
|------|--------|
| `internal/config/auth.go` | `ManagedIdentity` field, env var, merge, `AgentScope` constant |
| `internal/infrastructure/infrastructure.go` | Constructor branching for db/storage/agent based on managed identity flag |

## Validation Criteria

- `go vet ./...` passes
- `go build ./cmd/server/` passes
- `go test ./tests/...` passes
- `auth_mode: "none"` behavior completely unchanged (no credential, no branching)
- `auth_mode: "azure"` + `managed_identity: false` uses existing connection string constructors
- `auth_mode: "azure"` + `managed_identity: true` uses credential-based constructors + bearer agent factory
