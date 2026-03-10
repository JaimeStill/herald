# 107 - Add agent token provider for AI Foundry bearer auth

## Context

Part of Objective #97 (Managed Identity for Azure Services). When Herald runs with managed identity enabled, workflow agents need fresh Entra bearer tokens instead of a static API key. This task adds a `NewAgent` factory closure to `workflow.Runtime` that encapsulates auth-mode-aware agent creation, keeping workflow nodes clean.

## Architecture Approach

Replace `Runtime.Agent` (used as a template for `agent.New`) with a `NewAgent func(ctx context.Context) (agent.Agent, error)` factory closure built at the infrastructure/wiring layer. The factory resolves the auth strategy once at cold start:

- **API key mode** (nil credential): closure captures config by value, returns `agent.New(&cfg)` — token already in config from startup
- **Bearer mode** (non-nil credential): closure captures config + credential, acquires a cached token via Azure SDK, creates a stack-local `ProviderConfig` copy with the token injected (avoids concurrent map writes from errgroup goroutines), returns `agent.New(&cfg)`

`workflow.Runtime` retains `ModelName` and `ProviderName` strings for classification DB records (replacing direct `Agent.Model.Name` / `Agent.Provider.Name` access).

## Implementation

### Step 1: Update workflow.Runtime

**File:** `internal/workflow/runtime.go`

- Replace `Agent gaconfig.AgentConfig` with:
  - `NewAgent func(ctx context.Context) (agent.Agent, error)` — factory closure
  - `ModelName string` — for classification DB records
  - `ProviderName string` — for classification DB records
- Remove `gaconfig` import, add `context` and agent imports

### Step 2: Replace agent.New calls in workflow nodes

**File:** `internal/workflow/classify.go` (line 82)
- Replace `agent.New(&rt.Agent)` with `rt.NewAgent(gctx)`
- Remove `agent` import

**File:** `internal/workflow/enhance.go` (line 109)
- Replace `agent.New(&rt.Agent)` with `rt.NewAgent(gctx)`
- Remove `agent` import

**File:** `internal/workflow/finalize.go` (line 48)
- Replace `agent.New(&rt.Agent)` with `rt.NewAgent(ctx)`
- Remove `agent` import

### Step 3: Update classification repository metadata access

**File:** `internal/classifications/repository.go`
- Update `New()` signature: replace `agent gaconfig.AgentConfig` with `newAgent func(ctx context.Context) (agent.Agent, error)`, `modelName string`, `providerName string`
- Set `NewAgent`, `ModelName`, `ProviderName` on `workflow.Runtime`
- Update upsert args (lines 166-167): `r.rt.Agent.Model.Name` → `r.rt.ModelName`, `r.rt.Agent.Provider.Name` → `r.rt.ProviderName`
- Remove `gaconfig` import

### Step 4: Build factory closure in infrastructure wiring

**File:** `internal/infrastructure/infrastructure.go`
- Add `NewAgent func(ctx context.Context) (agent.Agent, error)` field to `Infrastructure`
- Build the factory in `New()`:
  - Current behavior (no managed identity): captures `cfg.Agent` by value, returns `agent.New(&cfg)`
  - Bearer path will be wired in issue #108 when `managed_identity` flag is added

**File:** `internal/api/runtime.go`
- Thread `infra.NewAgent` through when constructing the inner `Infrastructure` in `NewRuntime`

**File:** `internal/api/domain.go`
- Update `classifications.New()` call: pass `runtime.NewAgent`, `runtime.Agent.Model.Name`, `runtime.Agent.Provider.Name` instead of `runtime.Agent`

## Files Modified

| File | Change |
|------|--------|
| `internal/workflow/runtime.go` | Replace `Agent` config with `NewAgent` factory + metadata strings |
| `internal/workflow/classify.go` | `agent.New(&rt.Agent)` → `rt.NewAgent(gctx)` |
| `internal/workflow/enhance.go` | `agent.New(&rt.Agent)` → `rt.NewAgent(gctx)` |
| `internal/workflow/finalize.go` | `agent.New(&rt.Agent)` → `rt.NewAgent(ctx)` |
| `internal/classifications/repository.go` | Updated `New()` signature + metadata access |
| `internal/infrastructure/infrastructure.go` | Add `NewAgent` factory field |
| `internal/api/runtime.go` | Thread `NewAgent` |
| `internal/api/domain.go` | Pass factory + metadata to `classifications.New` |

## Validation Criteria

- [ ] `go vet ./...` passes
- [ ] `go build ./cmd/server/` succeeds
- [ ] All workflow nodes use `rt.NewAgent(ctx)` — no direct `agent.New` calls in workflow package
- [ ] Nil credential preserves existing api_key behavior (no functional change)
- [ ] `workflow.Runtime` has no `AgentConfig` dependency — only factory + metadata strings
