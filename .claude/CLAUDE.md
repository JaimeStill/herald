# Herald

Go web service for classifying DoD PDF documents' security markings using Azure AI Foundry GPT vision models. See `_project/README.md` for full architecture and roadmap.

## Reference Project

The `~/code/agent-lab` project is available for reference. Herald's architecture, package structure, and domain patterns (System/Repository/Handler/Mapping/Errors) are adapted from agent-lab. Consult it for proven patterns when implementing new domains or infrastructure that have not yet been adapted to Herald.

## Architecture

Herald follows the Layered Composition Architecture (LCA) from agent-lab: cold start (config load, subsystem creation) → hot start (connections, HTTP listen) → graceful shutdown (reverse-order teardown).

### Package Structure

- `cmd/` — Entry points (`package main`)
- `internal/` — Private application packages (config, infrastructure, domain systems)
- `pkg/` — Reusable library packages (database, lifecycle, middleware, module, etc.)
- `web/` — Web client

### Configuration Pattern

Every config struct follows the three-phase finalize pattern:
1. `loadDefaults()` — hardcoded fallbacks
2. `loadEnv(env)` — environment variable overrides (env var names injected via `Env` struct)
3. `validate()` — validate final values

Public API: `Finalize(env)` and `Merge(overlay)`. Env var names are centralized in `internal/config/` and injected into sub-package `Finalize()` methods.

### Dependency Hierarchy

Lower-level packages (`pkg/`) define contracts (interfaces). Higher-level packages (`internal/`) implement them. Dependencies flow downward only.

## AI Responsibilities

### API Cartographer Maintenance

API Cartographer maintenance is an AI responsibility. After an implementation plan is accepted, the AI generates or updates the corresponding `_project/api/<group>/README.md` and `.http` file before transitioning control to the developer. See `.claude/skills/api-cartographer/SKILL.md` for conventions.

### Web Development Skill Maintenance

After implementing changes that revise or enhance the web development architecture (new component patterns, service infrastructure changes, design system updates, build system modifications), update `.claude/skills/web-development/SKILL.md` and its `references/` to reflect current conventions. The skill is the source of truth for all subsequent web client work.

### Frontend Design

Use the `frontend-design` skill (Anthropic built-in) when planning web client view interfaces. This applies to any objective focused on building out view UIs (documents view, prompts view, review view, etc.). The frontend-design skill provides design quality guidance that complements the web-development skill's architectural patterns.

### Testing

All test authorship is an AI responsibility. Tests live in `tests/` mirroring the source structure. Black-box only (`package <name>_test`). Table-driven for parameterized cases. No test code in implementation guides.

### Documentation

Godoc comments on exported Go types, functions, and methods are an AI responsibility. JSDoc comments on exported TypeScript types, interfaces, service objects, and public functions are an AI responsibility. Both are added after implementation, not in guides.

## Development

### Build and Run

```bash
mise run dev       # go run ./cmd/server/
mise run build     # go build -o bin/server ./cmd/server
mise run test      # go test ./tests/...
mise run vet       # go vet ./...
```

### Local Infrastructure

```bash
docker compose up -d    # PostgreSQL (5432) + Azurite (10000)
docker compose down     # Stop and remove containers
```

### Environment Overlays

Config loading: `config.json` (base) → `config.<HERALD_ENV>.json` (overlay) → `secrets.json` (gitignored secrets) → `HERALD_*` env vars (overrides).

All env vars use the `HERALD_` prefix (e.g., `HERALD_SERVER_PORT`, `HERALD_DB_HOST`).

## Versioning

All version references across the project (ARM parameter files, config defaults, tags) must align with the current phase version target from `_project/phase.md`. ARM parameter files use four-part versions: `major.minor.patch.build`. The fourth position tracks deployment iterations — `0` for the phase release, `> 0` for dev builds (e.g., `0.4.0.0` is the v0.4.0 release, `0.4.0.3` is the third dev iteration).

## Go Conventions

- **Naming**: Short, singular, lowercase package names. No type stuttering.
- **Errors**: Lowercase, no punctuation, wrapped with context (`fmt.Errorf("operation failed: %w", err)`). Package-level errors in `errors.go` with `Err` prefix.
- **Modern idioms**: `sync.WaitGroup.Go()`, `for range n`, `min()`/`max()`, `errors.Join`.
- **Parameters**: More than two → use a struct.
- **Interfaces**: Define where consumed, not where implemented. Keep minimal.

## Web Client Conventions

See `.claude/skills/web-development/` for full conventions. Key points that are easy to miss:

- **Cascade layers**: `tokens, reset, base, theme, app`. The `base` layer (`design/core/base.css`) holds cross-cutting primitives like the universal `scrollbar-gutter: stable` rule.
- **Scroll containers**: Never declare `overflow-y: auto` (or `overflow-x: auto`) directly in a component's CSS. Import `@styles/scroll.module.css` and apply `.scroll-y` / `.scroll-x` in the template. The utility bundles `overflow`, `scrollbar-gutter: stable`, and axis padding — layout still lives in the component's own CSS (`flex: 1; min-height: 0;`, `max-height: ...`, etc.).
- **Flex sizing for scroll**: A scroll container still needs `flex: 1; min-height: 0;` from its parent flex column; the utility does not fix missing flex constraints.
- **Shared styles**: `@styles/*` for reusable modules (`badge`, `buttons`, `cards`, `inputs`, `labels`, `scroll`). Component `*.module.css` provides layout-specific overrides.

## Gotchas

- **Middleware order**: First-registered middleware runs outermost (CORS before Logger)
- **Module prefix stripping**: Inner mux sees paths WITHOUT the module prefix (e.g., `/agents/{id}` not `/api/agents/{id}`)
- **Shutdown hooks**: Must block on `<-lc.Context().Done()` before executing cleanup
