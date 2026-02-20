# 4 - Project Scaffolding, Configuration, Lifecycle, and HTTP Infrastructure

## Summary

Established Herald from zero Go source code to a running web service skeleton. Implemented the full foundation: Go module, project structure, build tooling (mise + Docker Compose), configuration system (TOML + env overlays + env var overrides), lifecycle coordinator, OpenAPI spec infrastructure, HTTP infrastructure (middleware, module routing, response helpers, route registration), and Scalar API documentation UI with embedded Bun + Vite build assets.

## Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Storage backend | Azure Blob Storage (Azurite emulator for local dev) | Aligns with DoD deployment target; Azurite provides parity |
| Scalar version | Pinned to 1.43.2 (exact) | Newer versions include @scalar/agent-chat which produces extra Vite chunks with no clean opt-out |
| Web build pipeline | Bun + Vite | Vite needed for Phase 3 Lit SPA; Bun as fast package manager/runtime |
| Scalar assets | Embedded via go:embed (Vite build) | No CDN dependency; assets compiled into binary |
| Docker volumes | Named volumes with driver_opts bind to ${HOME}/storage/herald/{store} | Persistent local storage with explicit host paths |
| Azurite account | Custom account name `heraldstore` via AZURITE_ACCOUNTS | Clean separation from default devstoreaccount1 |
| OpenAPI responsibility | AI maintains schema after plan approval | Convention established in CLAUDE.md; enables Scalar testing without curl |
| Logging | Direct slog.New() | Defer dedicated logging package unless needed |

## Files Modified

### Scaffolding
- `go.mod`, `go.sum`, `.gitignore`
- `doc.go` stubs for future packages (12 files)

### Build Tooling
- `.mise.toml`, `docker-compose.yml`, `compose/postgres.yml`, `compose/azurite.yml`, `config.toml`

### Configuration
- `pkg/database/config.go`, `pkg/storage/config.go`, `pkg/middleware/config.go`, `pkg/openapi/config.go`
- `internal/config/config.go`, `internal/config/server.go`, `internal/config/api.go`

### Lifecycle
- `pkg/lifecycle/lifecycle.go`

### OpenAPI
- `pkg/openapi/types.go`, `pkg/openapi/spec.go`, `pkg/openapi/components.go`, `pkg/openapi/json.go`

### HTTP Infrastructure
- `pkg/handlers/handlers.go`
- `pkg/routes/route.go`, `pkg/routes/group.go`
- `pkg/middleware/middleware.go`, `pkg/middleware/cors.go`, `pkg/middleware/logger.go`
- `pkg/module/module.go`, `pkg/module/router.go`

### Web / Scalar
- `web/package.json`, `web/tsconfig.json`, `web/vite.config.ts`, `web/vite.client.ts`
- `web/scalar/app.ts`, `web/scalar/client.config.ts`, `web/scalar/index.html`, `web/scalar/scalar.go`
- `web/scalar/scalar.js`, `web/scalar/scalar.css` (build artifacts)

### Walking Skeleton
- `cmd/server/main.go`

### Project Infrastructure
- `.claude/CLAUDE.md`

### Tests
- `tests/config/config_test.go`, `tests/database/config_test.go`, `tests/storage/config_test.go`
- `tests/lifecycle/lifecycle_test.go`, `tests/openapi/openapi_test.go`, `tests/handlers/handlers_test.go`
- `tests/routes/routes_test.go`, `tests/middleware/middleware_test.go`, `tests/module/module_test.go`

## Patterns Established

- **Three-phase config finalization**: `loadDefaults()` → `loadEnv(env)` → `validate()` with `Env` struct injection
- **Config overlay chain**: `config.toml` (base) → `config.<HERALD_ENV>.toml` (overlay) → `HERALD_*` env vars
- **Module routing**: Router dispatches by first path segment; Module strips prefix before delegating to inner mux
- **Middleware reverse-order application**: First-registered runs outermost
- **Route group OpenAPI integration**: Groups register handlers and populate OpenAPI spec in a single pass
- **Embedded web assets**: Bun + Vite builds to Go `embed.FS` directory; `mise run dev` triggers web build before `go run`

## Validation Results

- `go build ./...` — pass
- `go vet ./...` — pass
- `go mod tidy` — no changes
- `go test ./tests/...` — 63 tests, all pass
- `mise run dev` — server starts, loads config, waits for SIGINT
