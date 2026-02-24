# 29 - Agent Configuration and Infrastructure Wiring

## Summary

Migrated Herald's config system from TOML to JSON and added go-agents `AgentConfig` integration. Config structs across 7 files had tags changed from `toml:` to `json:`. The `pelletier/go-toml/v2` dependency was removed in favor of stdlib `encoding/json`. A new `internal/config/agent.go` provides `FinalizeAgent()` implementing Herald's three-phase finalize pattern (defaults, env overrides, validation) for the external go-agents type via standalone functions. Infrastructure validates agent config at startup by creating a test agent via `agent.New()`.

## Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Agent config type | `gaconfig.AgentConfig` value type (not pointer) | Consistent with all other Config fields (ServerConfig, database.Config, etc.) |
| Agent finalize approach | Standalone functions in `internal/config/agent.go` | Can't add methods to external type; keeps Herald's finalize pattern without a wrapper type |
| Env var pattern | Constants directly in `agent.go` (no `Env` struct) | Internal package — no need for the injection indirection that `pkg/` configs use |
| Agent env vars | 7 vars covering provider, model, and provider options | Enables full environment-based overrides for deployment flexibility (Azure vs Ollama) |
| Agent defaults | Seed from `gaconfig.DefaultAgentConfig()` | Ollama defaults allow config-free local development |
| Token handling | Injected via env var when set, not validated by Herald | Provider-specific validation delegated to `agent.New()` — Azure requires token, Ollama does not |

## Files Modified

- `internal/config/config.go` — JSON loading, Agent field, FinalizeAgent call
- `internal/config/agent.go` — New file: FinalizeAgent, constants, defaults/env/validation
- `internal/config/server.go` — Tags only
- `internal/config/api.go` — Tags only
- `internal/infrastructure/infrastructure.go` — Agent field, agent.New() validation
- `pkg/database/config.go` — Tags only
- `pkg/storage/config.go` — Tags only
- `pkg/middleware/config.go` — Tags only
- `pkg/pagination/config.go` — Tags only
- `config.json` — New file (replaces config.toml)
- `config.toml` — Deleted
- `go.mod` / `go.sum` — go-agents added, go-toml removed
- `tests/config/config_test.go` — Rewritten: JSON format, 5 new agent tests
- `tests/api/api_test.go` — Added agent config to validConfig()
- `tests/infrastructure/infrastructure_test.go` — Added agent config to validConfig()
- `_project/README.md` — Updated config format, Infrastructure struct, agent config example, dependencies
- `.claude/CLAUDE.md` — Updated config overlay description

## Patterns Established

- **External type finalize**: When an external type needs Herald's three-phase finalize pattern, create standalone functions (not a wrapper type) in `internal/config/`. Use constants for env vars since they're internal.
- **Agent defaults via Merge**: `loadAgentDefaults` seeds from `DefaultAgentConfig()`, merges loaded values on top, then assigns back — preserving the "defaults first, then overrides" semantic.

## Validation Results

- `go vet ./...` — clean
- `go test ./tests/...` — 15 packages, all pass
- `go mod tidy` — no drift
- Zero `toml:` struct tags remaining
- Zero `go-toml` imports remaining
