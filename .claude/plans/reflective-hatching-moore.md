# Task 136 — Migrate to tau/agent and tau/orchestrate

## Context

Herald currently depends on `github.com/JaimeStill/go-agents@v0.4.0` and `github.com/JaimeStill/go-agents-orchestration@v0.3.3`. These have been forked into the `tailored-agentic-units` GitHub org as independent modules. The issue body assumed a trivial import-path find/replace, but exploration shows **substantive API divergence**. This plan captures that divergence and separates the work into (a) the implementation guide the developer executes, and (b) the AI-owned closeout work (tests, docs, CHANGELOG, skill sweep, PR).

### Divergence discovered

| Concern | go-agents (current) | tau (target) |
|---------|---------------------|--------------|
| Package layout | `pkg/agent`, `pkg/config` | Top-level `agent` + sibling module `protocol/config` |
| Module layout | Two modules | Seven modules (`agent`, `protocol`, `format`, `format/openai`, `provider`, `provider/azure`, `orchestrate`) plus `provider/ollama` for tests |
| Agent factory | `agent.New(cfg)` one-shot | `provider.Create(cfg.Provider)` + `format.Create(cfg.Format)` + `agent.New(cfg, provider, format)` |
| Provider/format registration | Built in | Requires `azure.Register()` + `openai.Register()` + `ollama.Register()` side-effect calls |
| Vision API | `Vision(ctx, promptString, []string{dataURI})` | `Vision(ctx, []protocol.Message, []format.Image, opts...)` |
| Chat API | `Chat(ctx, promptString)` | `Chat(ctx, []protocol.Message, opts...)` |
| Response text | `resp.Content()` method | `resp.Text()` method |
| Image payload | Base64 data URI string | `format.Image{Data: []byte, Format: "png"}` — base64 encoding handled inside the format package |
| Graph event constants | `observability.EventNodeStart`, `EventNodeComplete`, `EventStateCreate`, etc. | Moved to the `state` package (`state.EventNodeStart`, etc.); `observability.Event`, `EventType`, `Observer` stay in observability |
| `AgentConfig.Format` | Not present | New field; `"openai"` default; required to resolve format factory |

### Tagged versions to target (no `replace`)

- `github.com/tailored-agentic-units/agent v0.1.1`
- `github.com/tailored-agentic-units/orchestrate v0.1.0`
- `github.com/tailored-agentic-units/protocol v0.1.0`
- `github.com/tailored-agentic-units/format v0.1.0`
- `github.com/tailored-agentic-units/format/openai v0.1.0` (repo tag `openai/v0.1.0`)
- `github.com/tailored-agentic-units/provider v0.1.0`
- `github.com/tailored-agentic-units/provider/azure v0.1.0` (repo tag `azure/v0.1.0`)
- `github.com/tailored-agentic-units/provider/ollama v0.1.0` (repo tag `ollama/v0.1.0`) — needed because unit tests set `Provider.Name = "ollama"`

## Implementation Guide Scope (developer executes)

The implementation guide covers ONLY source code changes. Tests, documentation, CHANGELOG, and skill sweeps are handled by the AI after execution.

### Step 1 — `go.mod` / `go.sum`

Remove the two JaimeStill modules. Add the eight tau modules above as direct dependencies. `go mod tidy` reconciles transitive entries. No `replace` directives.

### Step 2 — `internal/config/config.go`

Swap `gaconfig "github.com/JaimeStill/go-agents/pkg/config"` → `gaconfig "github.com/tailored-agentic-units/protocol/config"`. No other changes — `AgentConfig` field still exists, `Merge` still works.

### Step 3 — `internal/config/agent.go`

Same import swap. The three-phase finalize (`loadAgentDefaults` → `loadAgentEnv` → `validateAgent`) remains unchanged. `gaconfig.DefaultAgentConfig()`, `gaconfig.ProviderConfig{}`, `gaconfig.ModelConfig{}` all exist under the new path with compatible signatures.

### Step 4 — `internal/infrastructure/infrastructure.go`

Three distinct changes:

1. **Provider/format registration.** Add a package-level `sync.Once` that calls `azure.Register()`, `ollama.Register()`, and `openai.Register()`. Invoke the once from `New` before any agent construction. This keeps registration self-contained so unit tests work without extra wiring.

2. **Field-type swap.** `Infrastructure.Agent gaconfig.AgentConfig` — update the import to the tau path; the type name and shape are unchanged.

3. **`NewAgent` factory rewrite.** Replace the one-shot `agent.New(&cfg.Agent)` with a three-step construction:

   ```go
   newAgent := func(ctx context.Context) (agent.Agent, error) {
       p, err := provider.Create(agentCfg.Provider)
       if err != nil {
           return nil, fmt.Errorf("create provider: %w", err)
       }
       f, err := format.Create(agentCfg.Format)
       if err != nil {
           return nil, fmt.Errorf("create format: %w", err)
       }
       return agent.New(&agentCfg, p, f), nil
   }
   ```

   Keep the per-request factory pattern — the `classify` and `enhance` errgroup goroutines depend on it, and tau's Azure provider caches managed-identity tokens internally, so the factory stays cheap.

   Replace the startup-time `agent.New(&cfg.Agent)` validation call with `newAgent(context.Background())` so configuration errors surface at cold start, matching current behavior.

### Step 5 — `internal/workflow/runtime.go`

Swap `"github.com/JaimeStill/go-agents/pkg/agent"` → `"github.com/tailored-agentic-units/agent"`. `Runtime.NewAgent func(context.Context) (agent.Agent, error)` signature stays.

### Step 6 — `internal/workflow/workflow.go`

- `gaoconfig "github.com/JaimeStill/go-agents-orchestration/pkg/config"` → `gaoconfig "github.com/tailored-agentic-units/orchestrate/config"`
- `"github.com/JaimeStill/go-agents-orchestration/pkg/state"` → `"github.com/tailored-agentic-units/orchestrate/state"`
- `state.New(nil)` works unchanged (tau accepts nil observer and substitutes `NoOpObserver`).
- `state.NewGraphWithDeps(cfg, observer, nil)` signature unchanged.
- `state.Not(predicate)` and `state.NewFunctionNode(...)` unchanged.

### Step 7 — `internal/workflow/classify.go`

- Update orchestrate import.
- Add tau imports: `"github.com/tailored-agentic-units/format"`, `"github.com/tailored-agentic-units/protocol"`.
- Drop `"github.com/JaimeStill/document-context/pkg/encoding"` and `"github.com/JaimeStill/document-context/pkg/document"` — the format package handles base64 encoding, so Herald only needs raw bytes.
- Rewrite the Vision call:

  ```go
  imgData, err := readPageImage(cs.Pages[i].ImagePath)
  if err != nil {
      return fmt.Errorf("page %d: %w", i+1, err)
  }

  resp, err := a.Vision(gctx,
      []protocol.Message{protocol.UserMessage(prompt)},
      []format.Image{{Data: imgData, Format: "png"}},
  )
  ...
  parsed, err := formatting.Parse[pageResponse](resp.Text())
  ```

- Replace the existing `encodePageImage` helper with a simpler `readPageImage(path string) ([]byte, error)` that wraps `os.ReadFile` with an error-context prefix. Keep it as a named helper since `enhance.go` uses it too.

### Step 8 — `internal/workflow/enhance.go`

Same pattern as classify.go — update imports, drop `document-context/pkg/encoding`, switch to `readPageImage` + Vision-with-messages/images. `document-context/pkg/document` and `pkg/image` imports remain (ImageMagick re-rendering still uses them).

### Step 9 — `internal/workflow/finalize.go`

- Update orchestrate import.
- Add `"github.com/tailored-agentic-units/protocol"`.
- Rewrite the Chat call:

  ```go
  resp, err := a.Chat(ctx, []protocol.Message{protocol.UserMessage(prompt)})
  ...
  parsed, err := formatting.Parse[finalizeResponse](resp.Text())
  ```

### Step 10 — `internal/workflow/observer.go`

- Replace `"github.com/JaimeStill/go-agents-orchestration/pkg/observability"` with both `"github.com/tailored-agentic-units/orchestrate/observability"` and `"github.com/tailored-agentic-units/orchestrate/state"`.
- `observability.Event` and `observability.Observer` types stay in observability.
- Event constants in the switch move to the `state` package:
  - `observability.EventNodeStart` → `state.EventNodeStart`
  - `observability.EventNodeComplete` → `state.EventNodeComplete`

### Step 11 — `internal/classifications/repository.go`

Swap the agent import path only. `agent.Agent` interface reference unchanged.

## AI-Owned Closeout Work (after developer execution)

These items are handled during Phases 5–8 of the task-execution workflow and must not appear in the implementation guide:

### Tests (Phase 5)

- `tests/workflow/observer_test.go` — swap the observability import for the tau `observability` + `state` pair; migrate every event constant reference (`EventNodeStart`, `EventNodeComplete`, `EventStateCreate`, `EventStateSet`, `EventGraphStart`, `EventGraphComplete`, `EventEdgeTransition`, `EventEdgeEvaluate`) to the `state` package. Keep `observability.Event`/`EventType` references.
- `tests/api/api_test.go` and `tests/infrastructure/infrastructure_test.go` — swap `gaconfig` to the tau protocol/config path; keep the `"ollama"` provider config (tau's ollama provider is registered inside `infrastructure.New`). The new `AgentConfig.Format` field defaults to `"openai"` via the finalize step, so test fixtures need no update.
- No new test files are required. Existing suites cover observer translation, graph wiring, and infrastructure composition.

### Documentation (Phase 7)

- `_project/README.md` — update the **Dependencies → Go Libraries (ecosystem)** bullets and the `Infrastructure` code block narrative to reference `tau/agent`, `tau/orchestrate`, `tau/format`, `tau/provider`.
- Godoc comments on any functions whose signatures changed (`readPageImage`, updated `NewAgent` factory logic, updated observer switch) — add/update as needed. No godoc in the implementation guide itself.
- `.claude/skills/` sweep — `grep -r "go-agents"` across the skills directory and update any canonical guidance. Archived files under `.claude/context/guides/.archive/` are historical and left alone.

### Closeout (Phase 8)

- **CHANGELOG.md** — prepend a `## v0.5.0-dev.132.136` section with a one-liner describing the migration (tag format `v<target>-dev.<objective>.<issue>`).
- **Session summary** at `.claude/context/sessions/136-tau-migration.md`.
- **Archive** the implementation guide to `.claude/context/guides/.archive/`.
- **Commit + PR** with `Closes #136` in the body.

## Verification (developer-run during Phase 6)

1. `go mod tidy` — module graph clean.
2. `mise run vet` — import/signature errors caught.
3. `mise run test` — all suites pass (observer, api, infrastructure).
4. `mise run dev` with Azure credentials — upload a known-classified test PDF through the web client, trigger classification, confirm the workflow produces a correct result end-to-end.
5. `grep -r "JaimeStill/go-agents" internal tests cmd pkg` returns nothing; `grep "replace " go.mod` returns nothing.

## Critical Files (implementation guide touches)

- `go.mod`, `go.sum`
- `internal/config/config.go`, `internal/config/agent.go`
- `internal/infrastructure/infrastructure.go`
- `internal/workflow/workflow.go`, `runtime.go`, `classify.go`, `enhance.go`, `finalize.go`, `observer.go`
- `internal/classifications/repository.go`

## Out of Scope

- Functional behavior changes (prompt wording, workflow topology, agent concurrency model) are untouched.
- Checkpointing and other orchestrate features Herald did not previously consume.
- Redesign of the per-request agent factory — tau supports the same pattern cheaply.
