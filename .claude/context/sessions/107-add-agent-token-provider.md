# 107 - Add agent token provider for AI Foundry bearer auth

## Summary

Replaced direct `agent.New(&rt.Agent)` calls in workflow nodes with a `NewAgent` factory closure on `workflow.Runtime`. The factory is built at the infrastructure layer during cold start, encapsulating auth-mode-aware agent creation. Currently wires the API key path; issue #108 will add the bearer token branch based on `managed_identity` flag. Also extracted `Model` and `Provider` string fields onto `Runtime` for classification DB record metadata, removing `workflow`'s dependency on `gaconfig.AgentConfig`.

## Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Factory closure vs TokenProvider field | Factory closure built at infrastructure layer | Resolves auth strategy once at cold start; workflow nodes just call `rt.NewAgent(ctx)` without auth awareness |
| Config cloning scope | Stack-local ProviderConfig + Options map clone per call (bearer path, wired in #108) | Avoids concurrent map writes from errgroup goroutines; minimal clone scope |
| Metadata access | `Model` and `Provider` string fields on Runtime | Extracted from config at wiring time; avoids Runtime depending on full AgentConfig |

## Files Modified

- `internal/workflow/runtime.go` — replaced `Agent gaconfig.AgentConfig` with `NewAgent` factory + `Model`/`Provider` strings
- `internal/workflow/classify.go` — `rt.NewAgent(gctx)` replaces `agent.New(&rt.Agent)`
- `internal/workflow/enhance.go` — same swap
- `internal/workflow/finalize.go` — same swap
- `internal/classifications/repository.go` — updated `New()` signature and metadata access
- `internal/infrastructure/infrastructure.go` — added `NewAgent` factory field, built closure in `New()`
- `internal/api/runtime.go` — threaded `NewAgent` through
- `internal/api/domain.go` — passes factory + metadata to `classifications.New()`
- `tests/infrastructure/infrastructure_test.go` — added `TestNewAgentFactory`

## Patterns Established

- **Agent factory closure**: Infrastructure builds `NewAgent func(ctx context.Context) (agent.Agent, error)` at cold start. Downstream consumers receive the factory, not raw config. This pattern scales to bearer auth (issue #108) by swapping the closure implementation without touching workflow code.

## Validation Results

- `go vet ./...` — passes
- `go build ./cmd/server/` — passes
- `go test ./tests/...` — all 20 packages pass
- No direct `agent.New` calls in `internal/workflow/`
- `workflow.Runtime` has no `gaconfig` dependency
