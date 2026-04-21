# 136 - Migrate to tau/agent and tau/orchestrate

## Summary

Replaced `github.com/JaimeStill/go-agents@v0.4.0` and `github.com/JaimeStill/go-agents-orchestration@v0.3.3` with the `tailored-agentic-units` tau module graph (`agent`, `orchestrate`, `protocol`, `format`, `format/openai`, `provider`, `provider/azure`, `provider/ollama`). The migration was substantially larger than the issue anticipated: tau restructured the package layout (`pkg/agent` + `pkg/config` → top-level `agent` + `protocol/config`), split one module into seven, introduced a required provider/format registration step, and changed the `Vision`/`Chat` method signatures to use typed `[]protocol.Message` + `[]format.Image` instead of raw string prompts and base64 data URIs.

## Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Agent factory shape | Keep per-request `NewAgent` factory; construct via `provider.Create` + `format.Create` + `agent.New` inside the closure | Preserves the existing errgroup-per-page concurrency model in classify/enhance without redesigning workflow concurrency. Tau's Azure provider caches managed-identity tokens internally, so per-request factory stays cheap. |
| Provider/format registration | `sync.Once` inside `infrastructure.New` that calls `azure.Register()` + `ollama.Register()` + `openai.Register()` | Self-contained: production and unit tests both wire up without extra setup. Idempotent across repeated `New` invocations in the test suite. |
| Vision message role | `protocol.UserMessage(prompt)` | Tau's openai format attaches images to the last message and requires string content. User role matches the prior go-agents behavior and is the standard shape for OpenAI vision requests. |
| Image payload | `format.Image{Data: rawBytes, Format: "png"}` | Tau's format package handles base64 encoding internally; Herald only provides raw PNG bytes. Eliminates the `document-context/pkg/encoding` dependency in the classify/enhance callers (init.go still uses `document-context` for PDF→PNG rendering). |
| Event constants | Swapped call sites from `observability.EventNode*` to `state.EventNode*` | Tau moved graph event constants (`EventNodeStart`, `EventNodeComplete`, `EventStateCreate`, etc.) from `observability` to `state`. `observability.Event`/`Observer`/`EventType` types still live in `observability`. |
| Test fixture `Format` field | Explicitly set `Format: "openai"` in infra and api test `validConfig()` | Those fixtures bypass `FinalizeAgent`, so the default merge that fills `Format` doesn't run. Setting it in the fixture is more localized than refactoring tests to round-trip through `config.Load`. |
| `TestAgentDefaults` base_url expectation | Now asserts empty `Provider.BaseURL` | Tau's `DefaultProviderConfig` intentionally dropped the hard-coded `http://localhost:11434` — providers auto-construct a default base URL at request time when unset. This is a real semantic change in defaults, not a test bug. |

## Files Modified

### Source
- `go.mod`, `go.sum` — swap two JaimeStill modules for eight tau modules
- `internal/config/config.go` — `gaconfig` → `tauconfig` import
- `internal/config/agent.go` — `gaconfig` → `tauconfig`, godoc refresh
- `internal/infrastructure/infrastructure.go` — register-once helper, tau imports, three-step `NewAgent` factory, validation-call refactor, godoc refresh
- `internal/workflow/workflow.go` — `gaoconfig` + `state` tau imports
- `internal/workflow/runtime.go` — agent import swap
- `internal/workflow/init.go` — orchestrate/state import swap
- `internal/workflow/classify.go` — drop `document-context/pkg/encoding`, add tau format/protocol, replace `encodePageImage` with `readPageImage`, rewrite Vision call
- `internal/workflow/enhance.go` — same pattern as classify
- `internal/workflow/finalize.go` — tau protocol import, rewrite Chat call, `resp.Content()` → `resp.Text()`
- `internal/workflow/observer.go` — split observability import, move event case constants to `state`, godoc refresh
- `internal/classifications/repository.go` — agent import swap

### Tests
- `tests/api/api_test.go` — tauconfig import, `Format: "openai"` fixture
- `tests/infrastructure/infrastructure_test.go` — tauconfig import, `Format: "openai"` fixture
- `tests/workflow/observer_test.go` — add state import, move all event constants to state package
- `tests/config/config_test.go` — update `TestAgentDefaults` base_url assertion + stale comment

### Docs
- `_project/README.md` — Phase 5 row (pre-existing edit, left intact), Dependencies → Go Libraries (ecosystem) rewritten for tau, Infrastructure struct type comment updated
- `CHANGELOG.md` — v0.5.0-dev.132.136 entry

## Patterns Established

- **Tau provider/format registration** — new convention: `sync.Once` inside `infrastructure.New` is the one place factories get wired. Any future tau provider/format addition goes in the same `registerAgentBackends` body.
- **Test fixture `Format` field** — unit tests that build `AgentConfig` literals must set `Format: "openai"` (or another registered format) explicitly since they bypass `FinalizeAgent`.
- **Vision/Chat call pattern** — prompt string + image path becomes `[]protocol.Message{protocol.UserMessage(prompt)}` + `[]format.Image{{Data: bytes, Format: "png"}}`; text extracted via `resp.Text()`, not `resp.Content()`.

## Validation Results

- `mise run vet` — clean
- `mise run test` — all 20 packages pass
- `mise run dev` end-to-end — three classification workflows ran successfully, including one that traversed the enhance node (confirmed by user)
