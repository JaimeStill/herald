# Issue #29 — Agent Configuration and Infrastructure Wiring

## Context

First sub-issue of Objective #24 (Phase 2). Herald's config system currently uses TOML format with the `pelletier/go-toml/v2` third-party dependency. This task migrates to JSON (stdlib `encoding/json`) and adds go-agents' `config.AgentConfig` type directly to Config and Infrastructure — no custom Herald agent config structs, no conversion layer. The agent token (`HERALD_AGENT_TOKEN`) is the only agent-specific env var; all other agent configuration lives in config.json.

## Step 1: Add go-agents dependency

```bash
go get github.com/JaimeStill/go-agents
```

Module: `github.com/JaimeStill/go-agents` (no version suffix — single-module repo).

## Step 2: Migrate struct tags from `toml` to `json` (7 files)

Mechanical find-replace of `toml:"x"` → `json:"x"` in every config struct:

| File | Fields |
|------|--------|
| `internal/config/config.go` | 6 fields on `Config` |
| `internal/config/server.go` | 5 fields on `ServerConfig` |
| `internal/config/api.go` | 4 fields on `APIConfig` |
| `pkg/database/config.go` | 10 fields on `Config` |
| `pkg/storage/config.go` | 3 fields on `Config` |
| `pkg/middleware/config.go` | 6 fields on `CORSConfig` |
| `pkg/pagination/config.go` | 2 fields on `Config` |

## Step 3: Switch config loading from TOML to JSON

In `internal/config/config.go`:

- Update constants:
  - `BaseConfigFile` → `"config.json"`
  - `OverlayConfigPattern` → `"config.%s.json"`
- In `load()` function: replace `toml.Unmarshal` → `json.Unmarshal`
- Replace import `github.com/pelletier/go-toml/v2` → `encoding/json`

## Step 4: Add Agent field to Config

In `internal/config/config.go`:

- Add import: `goconfig "github.com/JaimeStill/go-agents/pkg/config"`
- Add constant: `EnvHeraldAgentToken = "HERALD_AGENT_TOKEN"`
- Add field to Config: `Agent *goconfig.AgentConfig \`json:"agent"\``
- In `Merge()`: add agent merge logic (delegate to go-agents `AgentConfig.Merge()`)
- In `finalize()` after existing validation, add agent finalization:
  1. Validate `Agent` is not nil (fail fast — agent is required for classification)
  2. If `HERALD_AGENT_TOKEN` env var is set, ensure `Agent.Provider.Options` map is initialized and inject the token value as `Options["token"]`
  3. No token validation here — `agent.New()` in Infrastructure handles provider-specific validation (Azure requires token, Ollama does not)

## Step 5: Add Agent field to Infrastructure

In `internal/infrastructure/infrastructure.go`:

- Add import: `goconfig "github.com/JaimeStill/go-agents/pkg/config"` and `"github.com/JaimeStill/go-agents/pkg/agent"`
- Add field: `Agent *goconfig.AgentConfig`
- In `New()`: store `cfg.Agent` on Infrastructure, validate via `agent.New(cfg.Agent)` (discard the returned agent — this validates the full pipeline: provider creation, option extraction, token presence)

## Step 6: Create config.json (replace config.toml)

Translate existing `config.toml` to `config.json` and add the `agent` section in go-agents' native format:

```json
{
  "shutdown_timeout": "30s",
  "version": "0.2.0",
  "server": { ... },
  "database": { ... },
  "storage": { ... },
  "api": { ... },
  "agent": {
    "name": "herald-classifier",
    "provider": {
      "name": "azure",
      "base_url": "https://placeholder.openai.azure.com",
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
          "temperature": 0.1
        }
      }
    }
  }
}
```

Token is NOT in config.json — injected via `HERALD_AGENT_TOKEN` env var.

## Step 7: Remove go-toml dependency

```bash
go mod tidy
```

This removes `pelletier/go-toml/v2` (no longer imported anywhere). Delete `config.toml`.

## Validation Criteria

- [ ] All struct tags migrated from `toml` to `json`
- [ ] `pelletier/go-toml/v2` dependency removed, `encoding/json` used
- [ ] `config.json` replaces `config.toml` with overlay pattern `config.<env>.json`
- [ ] Existing tests pass with JSON config format
- [ ] `Config.Agent` is `*config.AgentConfig` (go-agents type)
- [ ] `HERALD_AGENT_TOKEN` injected into `Provider.Options["token"]` when env var is set
- [ ] `Infrastructure.Agent` validated via `agent.New()` during `New()`
- [ ] `go vet ./...` passes
- [ ] `go mod tidy` produces no changes

## Key Files

| File | Change |
|------|--------|
| `internal/config/config.go` | Tags, imports, loading, Agent field, finalize |
| `internal/config/server.go` | Tags only |
| `internal/config/api.go` | Tags only |
| `pkg/database/config.go` | Tags only |
| `pkg/storage/config.go` | Tags only |
| `pkg/middleware/config.go` | Tags only |
| `pkg/pagination/config.go` | Tags only |
| `internal/infrastructure/infrastructure.go` | Agent field, validation |
| `config.json` | New file (replaces config.toml) |
| `config.toml` | Deleted |
