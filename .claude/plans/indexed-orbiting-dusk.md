# Objective #24: Agent Configuration and Database Schema

## Context

First objective in Phase 2 (Classification Engine, v0.2.0). Extends the Phase 1 foundation with agent configuration for Azure AI Foundry vision models and database schema for classification results and prompt overrides. All subsequent Phase 2 objectives depend on this work.

As part of this work, the config format migrates from TOML to JSON. This eliminates the `go-toml` dependency (replaced by stdlib `encoding/json`) and enables the agent config section to use go-agents `config.AgentConfig` directly — no custom Herald config structs, no conversion layer, no bridging.

## Sub-Issues

### 1. Agent configuration and Infrastructure wiring

**Labels:** `feature`
**Milestone:** `v0.2.0 - Classification Engine`

Migrates config from TOML to JSON, adds go-agents dependency, embeds `config.AgentConfig` directly on the Config struct, and stores it on Infrastructure. Agents are created per-request (future objectives), not at startup.

#### Step 1: Migrate config from TOML to JSON

**Struct tag changes (`toml:"x"` → `json:"x"`):**
- `internal/config/config.go` — Config struct
- `internal/config/server.go` — ServerConfig struct
- `internal/config/api.go` — APIConfig struct
- `pkg/database/config.go` — database.Config struct
- `pkg/storage/config.go` — storage.Config struct
- `pkg/middleware/config.go` — CORSConfig struct
- `pkg/pagination/config.go` — pagination.Config struct

**Loading code (`internal/config/config.go`):**
- Replace `toml.Unmarshal` → `json.Unmarshal`
- Replace `"github.com/pelletier/go-toml/v2"` import → `"encoding/json"`
- `BaseConfigFile` → `"config.json"`
- `OverlayConfigPattern` → `"config.%s.json"`

**Config file:**
- `config.toml` → `config.json` (translate content to JSON)

**Tests (`tests/config/config_test.go`):**
- Convert embedded TOML strings to JSON
- Update filenames from `config.toml` → `config.json`, `config.staging.toml` → `config.staging.json`
- Update `TestLoadInvalidConfig` to use invalid JSON

**Dependency:**
- `go get -u` then `go mod tidy` removes `pelletier/go-toml/v2`

**Documentation updates:**
- `_project/README.md` — update go-toml reference, config file references
- `.claude/CLAUDE.md` — update config file references and overlay description

#### Step 2: Add agent config using go-agents types

**Scope:**
- `go get github.com/JaimeStill/go-agents`
- Add `Agent *goconfig.AgentConfig` field to `Config` struct (`json:"agent"`)
- Add agent section to `config.json` in go-agents native JSON format
- Apply `HERALD_AGENT_TOKEN` env var post-load (sets `Provider.Options["token"]`)
- Add `Agent *goconfig.AgentConfig` field to `Infrastructure`, populated from `Config.Agent` during `New()`
- Validate at startup by calling `agent.New()` with the config (discard test agent)

**Agent section in config.json** (mirrors go-agents JSON format exactly):
```json
{
  "agent": {
    "name": "herald-classifier",
    "provider": {
      "name": "azure",
      "base_url": "https://...",
      "options": {
        "deployment": "gpt-5-mini",
        "api_version": "2024-12-01-preview",
        "auth_type": "api_key"
      }
    },
    "model": {
      "name": "gpt-5-mini",
      "capabilities": {
        "vision": {
          "max_tokens": 4096,
          "temperature": 0.1,
          "vision_options": {
            "detail": "high"
          }
        }
      }
    }
  }
}
```

**Env var:** `HERALD_AGENT_TOKEN` — applied in `Config.finalize()` after JSON load. Sets `Agent.Provider.Options["token"]`. Never in config files.

**Per-request agent creation (implemented in future objectives):**
```
Infrastructure.Agent (*config.AgentConfig)
  → clone via go-agents Merge
  → merge request-body overrides
  → agent.New(merged)
  → use → discard
```

**Key files:**
- `internal/config/config.go` — add Agent field, apply token env var in finalize
- `internal/infrastructure/infrastructure.go` — add Agent field, validate in New()
- `config.json` — add agent section

**Acceptance criteria:**
- [ ] All struct tags migrated from `toml` to `json`
- [ ] `pelletier/go-toml/v2` dependency removed, `encoding/json` used
- [ ] `config.json` replaces `config.toml` with overlay pattern `config.<env>.json`
- [ ] Existing tests pass with JSON config format
- [ ] `Config.Agent` is `*config.AgentConfig` (go-agents type) — no Herald wrapper structs
- [ ] `HERALD_AGENT_TOKEN` applied post-load; server fails fast when missing
- [ ] `Infrastructure.Agent` validated via `agent.New()` during `New()`
- [ ] Tests cover agent config loading, token env var, validation failure

**Dependencies:** None

---

### 2. Classification engine database migration

**Labels:** `feature`
**Milestone:** `v0.2.0 - Classification Engine`

Creates the `classifications` and `prompts` tables in a single migration.

**Scope:**
- `000002_classification_engine.up.sql` and `.down.sql` in `cmd/migrate/migrations/`

**classifications table:**
```
id              UUID PK DEFAULT gen_random_uuid()
document_id     UUID NOT NULL UNIQUE FK → documents(id) ON DELETE CASCADE
classification  TEXT NOT NULL
confidence      TEXT NOT NULL CHECK ('HIGH','MEDIUM','LOW')
markings_found  JSONB NOT NULL DEFAULT '[]'
rationale       TEXT NOT NULL
classified_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
model_name      TEXT NOT NULL
provider_name   TEXT NOT NULL
validated_by    TEXT
validated_at    TIMESTAMPTZ
```

**prompts table:**
```
id            UUID PK DEFAULT gen_random_uuid()
name          TEXT NOT NULL UNIQUE
stage         TEXT NOT NULL CHECK ('init','classify','enhance')
system_prompt TEXT NOT NULL
description   TEXT
```

**Key files:**
- `cmd/migrate/migrations/000002_classification_engine.up.sql` — new
- `cmd/migrate/migrations/000002_classification_engine.down.sql` — new

**Acceptance criteria:**
- [ ] Migration applies and reverts cleanly
- [ ] 1:1 document-classification enforced via UNIQUE constraint + ON DELETE CASCADE
- [ ] Confidence CHECK-constrained to HIGH/MEDIUM/LOW
- [ ] Stage CHECK-constrained to init/classify/enhance
- [ ] Indexes on classification, confidence, classified_at DESC, stage

**Dependencies:** None (independent of sub-issue 1)

---

## Dependency Graph

```
Sub-issue 1 (Config Migration + Agent Config + Infra)    independent
Sub-issue 2 (Migration)                                  independent
```

Both can be developed in parallel.

## Architecture Decisions

1. **JSON config format**: Replaces TOML. Eliminates third-party dependency (`encoding/json` is stdlib). Enables go-agents `config.AgentConfig` to be embedded directly on Config — no custom Herald config structs, no conversion layer. Config keys use snake_case matching go-agents' json tags.

2. **Per-request agent creation**: Agents are created per-request from `Infrastructure.Agent` (go-agents `*config.AgentConfig`). Base config is cloned via `Merge()`, merged with request overrides, then `agent.New()` creates the agent. Agent creation is cheap (struct allocation only). This makes the full config surface overridable at runtime without custom merge logic.

3. **Go-agents config directly on Config/Infrastructure**: No Herald-specific agent config types. `Config.Agent` and `Infrastructure.Agent` are both `*config.AgentConfig` from go-agents. The JSON config file's agent section uses go-agents' native format.

4. **Startup validation via agent.New()**: `infrastructure.New()` calls `agent.New()` against the stored config to validate the full pipeline (provider creation, option validation). The test agent is discarded.

5. **Token env var only**: `HERALD_AGENT_TOKEN` is the only agent-specific env var (sensitive, never in config files). All other agent configuration via config.json overlays (`config.<env>.json`), suitable for Kubernetes ConfigMap mounts.

6. **Auth type configurable**: `auth_type` in provider options — `"api_key"` (Azure API key via `api-key` header) or `"bearer"` (Entra ID token via `Authorization: Bearer` header). Set in config.json, not env var.

7. **Provider name**: go-agents registers `"azure"`, not `"azure-ai-foundry"`. Config uses `"azure"`.

8. **Only go-agents added**: go-agents-orchestration deferred to later objectives.

9. **`classification` column unconstrained**: Security marking values may include unexpected levels. Unlike `confidence` (known enum), not CHECK-constrained.

10. **Deployment defaults to model name**: Azure deployments typically named after their model. No separate config field.

## Verification

- `go build ./...` and `go vet ./...` pass
- `mise run test` passes (all existing tests updated for JSON format)
- `mise run dev` starts with valid agent config + `HERALD_AGENT_TOKEN`
- Server fails fast with clear error when token missing
- Migration up/down cycles cleanly against local PostgreSQL
