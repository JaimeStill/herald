# Objective #24 — Agent Configuration and Database Schema

**Phase:** [Phase 2 — Classification Engine](phase.md)
**Issue:** [#24](https://github.com/JaimeStill/herald/issues/24)
**Milestone:** v0.2.0 - Classification Engine

## Scope

Extend Herald's configuration system to support Azure AI Foundry agent configuration and create database migrations for classification results and prompt overrides. This is the foundation objective for Phase 2 — all subsequent objectives depend on this work.

As part of this objective, the config format migrates from TOML to JSON. This eliminates the `go-toml` third-party dependency in favor of stdlib `encoding/json` and enables the agent config section to use go-agents' `config.AgentConfig` type directly — no custom Herald config structs, no conversion layer.

## Sub-Issues

| # | Sub-Issue | Status | Dependencies |
|---|-----------|--------|--------------|
| [#29](https://github.com/JaimeStill/herald/issues/29) | Agent configuration and Infrastructure wiring | Open | — |
| [#30](https://github.com/JaimeStill/herald/issues/30) | Classification engine database migration | Open | — |

Both sub-issues are independent and can be developed in parallel.

## Architecture Decisions

1. **JSON config format**: Replaces TOML. Eliminates third-party dependency (`encoding/json` is stdlib). Enables go-agents `config.AgentConfig` to be embedded directly on Config — no custom Herald config structs, no conversion layer. Config keys use snake_case matching go-agents' json tags.

2. **Per-request agent creation**: Agents are created per-request from `Infrastructure.Agent` (go-agents `*config.AgentConfig`). Base config is cloned via `Merge()`, merged with request overrides, then `agent.New()` creates the agent. Agent creation is cheap (struct allocation only). This makes the full config surface overridable at runtime without custom merge logic.

3. **Go-agents config directly on Config/Infrastructure**: No Herald-specific agent config types. `Config.Agent` and `Infrastructure.Agent` are both `*config.AgentConfig` from go-agents. The JSON config file's agent section uses go-agents' native format.

4. **Startup validation via agent.New()**: `infrastructure.New()` calls `agent.New()` against the stored config to validate the full pipeline (provider creation, option validation). The test agent is discarded.

5. **Token env var only**: `HERALD_AGENT_TOKEN` is the only agent-specific env var (sensitive, never in config files). All other agent configuration via config.json overlays (`config.<env>.json`), suitable for Kubernetes ConfigMap mounts.

6. **Auth type configurable**: `auth_type` in provider options — `"api_key"` (Azure API key via `api-key` header) or `"bearer"` (Entra ID token via `Authorization: Bearer` header). Set in config.json, not env var.

7. **Provider name**: go-agents registers `"azure"`, not `"azure-ai-foundry"`. Config uses `"azure"`.

8. **Only go-agents added**: go-agents-orchestration deferred to Objective #26.

9. **`classification` column unconstrained**: Security marking values may include unexpected levels. Unlike `confidence` (known enum), not CHECK-constrained.

10. **Deployment defaults to model name**: Azure deployments typically named after their model. No separate config field.
