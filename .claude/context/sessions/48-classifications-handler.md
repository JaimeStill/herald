# 48 - Classifications Handler, API Wiring, and API Cartographer Docs

## Summary

Built the HTTP layer for the classifications domain: a Handler struct with 8 endpoints, API module wiring with internalized workflow runtime construction, and API Cartographer documentation. Also added secrets.json config support, Azure AI Foundry provisioning scripts, and performed end-to-end classification testing. This completes the full classifications vertical from Objective #27.

## Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Workflow runtime ownership | Internalized in `classifications.New`, not constructed in `api.NewDomain` | Workflow is a sub-dependency of classifications, not a peer. Encapsulating the runtime construction keeps workflow composition from leaking into the API composition root. |
| Logger differentiation | `logger.With("workflow", "classify")` for runtime, `logger.With("system", "classifications")` for repo | Differentiates workflow operations from CRUD operations in logs without duplication. |
| `api/domain.go` workflow import | Removed — `api` no longer imports `workflow` | Classifications owns the runtime; the composition root passes raw infrastructure deps. |
| Handler constructor | `NewHandler(sys, logger, pagination)` matching prompts pattern | No domain-specific config (unlike documents' `maxUploadSize`). |
| Classify response status | 201 Created | New classification result is created (upsert semantics). |
| Secrets config | `secrets.json` merged in config pipeline after overlay, before env vars | Same JSON structure and `Merge` semantics as existing config — no new dependencies. Gitignored for local secret storage. |

## Files Modified

- `internal/classifications/system.go` — added `Handler()` to System interface
- `internal/classifications/repository.go` — refactored `New` to accept raw deps and internalize workflow.Runtime; added `Handler()` method
- `internal/classifications/handler.go` — new file with 8 HTTP endpoints
- `internal/api/domain.go` — added Classifications field, removed workflow import, passes raw deps
- `internal/api/routes.go` — registered classifications route group
- `internal/api/runtime.go` — added missing `Agent` field to `NewRuntime` Infrastructure struct literal
- `internal/config/config.go` — added `SecretsConfigFile` constant and secrets merge step in `Load()`
- `.gitignore` — added `secrets.json`
- `.claude/settings.json` — added `Read(secrets.json)` to deny list
- `config.json` — updated agent configuration for Azure AI Foundry deployment
- `scripts/` — new Azure AI Foundry provisioning scripts (create, destroy, query, defaults, components)
- `tests/classifications/handler_test.go` — new file with 35 handler test cases
- `_project/api/classifications/README.md` — new API Cartographer docs
- `_project/api/classifications/classifications.http` — new REST client file

## Bugs Found

- **Missing Agent in API Runtime**: `api.NewRuntime` constructed a new `Infrastructure` struct for logger scoping but omitted the `Agent` field, causing a nil pointer dereference when the classify endpoint tried to create an agent. Fixed by adding `Agent: cfg.Agent` to the struct literal.

## Patterns Established

- **Layered runtime convention**: Domains that need cross-cutting infrastructure (like workflow) receive raw dependencies and construct internal runtimes. The API composition root passes infrastructure and peer systems; domains own their internal wiring. This prevents dependency duplication and keeps workflow structure encapsulated.
- **Workflow logger context**: Workflow runtimes use `logger.With("workflow", "<name>")` to differentiate from the owning domain's system logger.
- **Secrets config pipeline**: `config.json` → `config.<env>.json` → `secrets.json` → `HERALD_*` env vars. Same merge semantics throughout, env vars always win.

## Follow-up Issues

- [#51 — Parallelize classify and enhance workflow nodes](https://github.com/JaimeStill/herald/issues/51): Sequential page processing takes ~13s per LLM call. Both nodes should use bounded `errgroup` concurrency with isolated per-page classification, deferring synthesis to finalize.

## Validation Results

- `go vet ./...` passes
- All 18 test suites pass (35 new handler tests + existing tests)
- `go mod tidy` produces no changes
- End-to-end classification tested: upload → classify → list/find → validate/update → delete flow verified against live Azure AI Foundry deployment (gpt-5-mini)
