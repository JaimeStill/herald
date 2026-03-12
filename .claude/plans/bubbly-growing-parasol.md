# #124 — Make AgentScope Configurable for Azure Government

## Context

`AgentScope` is hardcoded as `https://cognitiveservices.azure.com/.default` in `pkg/auth/config.go`. Azure Government uses `https://cognitiveservices.azure.us/.default`. This is the only application code change needed for Gov cloud support — all other config flows through `HERALD_*` env vars set by Bicep at deployment time.

## Approach

Convert the `AgentScope` constant to a configurable field on `auth.Config`, following the established config pattern (field, default, env override, merge).

## Changes

### 1. `pkg/auth/config.go`

- Remove the `AgentScope` constant
- Add `AgentScope string` field to `Config` struct (json: `"agent_scope"`)
- Add `AgentScope string` field to `Env` struct
- Set default `https://cognitiveservices.azure.com/.default` in `loadDefaults()`
- Add env override in `loadEnv()`
- Add merge logic in `Merge()`

### 2. `internal/config/config.go`

- Add `AgentScope: "HERALD_AUTH_AGENT_SCOPE"` to the `authEnv` var

### 3. `internal/infrastructure/infrastructure.go`

- In `newAgentFactory`, replace `auth.AgentScope` with `agentScope string` parameter
- Pass `cfg.Auth.AgentScope` from `New()` into `newAgentFactory()`

## Validation

- `go vet ./...`
- `go build ./cmd/server/`
- `go test ./tests/...`
