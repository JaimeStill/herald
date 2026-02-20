# Herald

Go web service for classifying DoD PDF documents' security markings using Azure AI Foundry GPT vision models. See `_project/README.md` for full architecture and roadmap.

## Architecture

Herald follows the Layered Composition Architecture (LCA) from agent-lab: cold start (config load, subsystem creation) → hot start (connections, HTTP listen) → graceful shutdown (reverse-order teardown).

### Package Structure

- `cmd/` — Entry points (`package main`)
- `internal/` — Private application packages (config, infrastructure, domain systems)
- `pkg/` — Reusable library packages (database, lifecycle, middleware, module, etc.)
- `workflow/` — Classification workflow definition (Phase 2)
- `web/` — Web client and Scalar API docs

### Configuration Pattern

Every config struct follows the three-phase finalize pattern:
1. `loadDefaults()` — hardcoded fallbacks
2. `loadEnv(env)` — environment variable overrides (env var names injected via `Env` struct)
3. `validate()` — validate final values

Public API: `Finalize(env)` and `Merge(overlay)`. Env var names are centralized in `internal/config/` and injected into sub-package `Finalize()` methods.

### Dependency Hierarchy

Lower-level packages (`pkg/`) define contracts (interfaces). Higher-level packages (`internal/`) implement them. Dependencies flow downward only.

## AI Responsibilities

### OpenAPI Schema Maintenance

OpenAPI schema maintenance is an AI responsibility. After an implementation plan is accepted, the AI completes all OpenAPI schema adjustments (operations, request/response schemas, parameters) before transitioning control to the developer. Domain handlers define their OpenAPI specs alongside route definitions.

### Testing

All test authorship is an AI responsibility. Tests live in `tests/` mirroring the source structure. Black-box only (`package <name>_test`). Table-driven for parameterized cases. No test code in implementation guides.

### Documentation

Godoc comments on exported types, functions, and methods are an AI responsibility. Added after implementation, not in guides.

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

Config loading: `config.toml` (base) → `config.<HERALD_ENV>.toml` (overlay) → `HERALD_*` env vars (overrides).

All env vars use the `HERALD_` prefix (e.g., `HERALD_SERVER_PORT`, `HERALD_DB_HOST`).

## Go Conventions

- **Naming**: Short, singular, lowercase package names. No type stuttering.
- **Errors**: Lowercase, no punctuation, wrapped with context (`fmt.Errorf("operation failed: %w", err)`). Package-level errors in `errors.go` with `Err` prefix.
- **Modern idioms**: `sync.WaitGroup.Go()`, `for range n`, `min()`/`max()`, `errors.Join`.
- **Parameters**: More than two → use a struct.
- **Interfaces**: Define where consumed, not where implemented. Keep minimal.

## Gotchas

- **Middleware order**: First-registered middleware runs outermost (CORS before Logger)
- **Module prefix stripping**: Inner mux sees paths WITHOUT the module prefix (e.g., `/agents/{id}` not `/api/agents/{id}`)
- **Shutdown hooks**: Must block on `<-lc.Context().Done()` before executing cleanup
