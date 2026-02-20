# Issue #7 — Infrastructure Assembly, API Module, and Server Entry Point

## Context

This is the final sub-issue of Objective #1 (Project Scaffolding, Configuration, and Service Skeleton). Issues #4, #5, and #6 established the foundational packages — lifecycle coordination, database toolkit, storage abstraction, middleware, module/routing, handlers, and configuration. This issue wires everything together into a running Go web service with health/readiness probes and graceful shutdown.

All patterns are adapted from `~/code/agent-lab` (infrastructure, API module, server lifecycle).

## Implementation

### Step 1: Add Pagination to APIConfig

**File: `internal/config/api.go`** (modify)

Add pagination config to `APIConfig` so the API Runtime can pass it to domain systems. Add the `paginationEnv` var, `Pagination pagination.Config` field, and wire it through `Finalize`, `Merge`, and `loadDefaults`.

**File: `config.toml`** (modify)

Add `[api.pagination]` section with `default_page_size = 20` and `max_page_size = 100`.

### Step 2: Infrastructure Assembly

**File: `internal/infrastructure/infrastructure.go`** (new, replaces `doc.go`)

Adapts `agent-lab/internal/infrastructure/infrastructure.go`:

- `Infrastructure` struct: `Lifecycle *lifecycle.Coordinator`, `Logger *slog.Logger`, `Database database.System`, `Storage storage.System`
- `New(cfg *config.Config) (*Infrastructure, error)` — creates lifecycle coordinator, logger (`slog.TextHandler` to stderr), database system, storage system. Cold start only (no connections).
- `Start() error` — calls `Database.Start(lc)` then `Storage.Start(lc)`. Hot start (registers lifecycle hooks).

Delete `internal/infrastructure/doc.go` stub.

### Step 3: API Module

Four files in `internal/api/`, replacing the `doc.go` stub:

**File: `internal/api/runtime.go`** (new)

- `Runtime` struct embedding `*infrastructure.Infrastructure` + `Pagination pagination.Config`
- `NewRuntime(cfg *config.Config, infra *Infrastructure) *Runtime` — creates module-scoped logger with `infra.Logger.With("module", "api")`

**File: `internal/api/domain.go`** (new)

- `Domain` struct — empty for now (domain systems added in Objectives #2 and #3)
- `NewDomain(runtime *Runtime) *Domain` — returns empty Domain

**File: `internal/api/routes.go`** (new)

- `registerRoutes(mux *http.ServeMux, spec *openapi.Spec, domain *Domain, cfg *config.Config)` — no-op for now (domain handlers registered when domains are implemented)

**File: `internal/api/api.go`** (new)

Adapts `agent-lab/internal/api/api.go`:

- `NewModule(cfg *config.Config, infra *Infrastructure) (*module.Module, error)` — creates Runtime, Domain, OpenAPI spec, mux, registers routes, marshals OpenAPI JSON, serves `/openapi.json`, creates module with middleware (CORS then Logger)

Delete `internal/api/doc.go` stub.

### Step 4: Server Entry Point

Three files in `cmd/server/`:

**File: `cmd/server/http.go`** (new)

Adapts `agent-lab/cmd/server/http.go`:

- `httpServer` struct wrapping `*http.Server` with logger and shutdown timeout
- `newHTTPServer(cfg *ServerConfig, handler http.Handler, logger *slog.Logger) *httpServer`
- `Start(lc *lifecycle.Coordinator) error` — ListenAndServe in goroutine, registers shutdown hook that blocks on `<-lc.Context().Done()` then calls `http.Server.Shutdown(ctx)`

**File: `cmd/server/modules.go`** (new)

Adapts `agent-lab/cmd/server/modules.go`:

- `Modules` struct with `API *module.Module`
- `NewModules(infra *Infrastructure, cfg *Config) (*Modules, error)` — creates API module
- `Mount(router *module.Router)` — mounts API module
- `buildRouter(infra *Infrastructure) *module.Router` — creates router, registers native health/readiness handlers:
  - `GET /healthz` → 200 JSON `{"status":"ok"}` (always)
  - `GET /readyz` → 200 JSON `{"status":"ready"}` when `infra.Lifecycle.Ready()` is true, 503 JSON `{"status":"not ready"}` otherwise

**File: `cmd/server/main.go`** (modify — replace current stub)

Adapts `agent-lab/cmd/server/main.go` + `server.go`:

- `Server` struct with infra, modules, http fields
- `NewServer(cfg) (*Server, error)` — cold start: create infrastructure, modules, router, HTTP server
- `Start() error` — hot start: `infra.Start()`, `http.Start(lc)`, async `WaitForStartup` logging
- `Shutdown(timeout) error` — delegates to lifecycle coordinator
- `main()` — Load config → NewServer → Start → wait for SIGINT/SIGTERM → Shutdown

## Files Summary

| File | Action | Reference |
|------|--------|-----------|
| `internal/config/api.go` | Modify | Add pagination config |
| `config.toml` | Modify | Add `[api.pagination]` section |
| `internal/infrastructure/doc.go` | Delete | Replaced by implementation |
| `internal/infrastructure/infrastructure.go` | Create | `agent-lab/internal/infrastructure/infrastructure.go` |
| `internal/api/doc.go` | Delete | Replaced by implementation |
| `internal/api/runtime.go` | Create | `agent-lab/internal/api/runtime.go` |
| `internal/api/domain.go` | Create | `agent-lab/internal/api/domain.go` |
| `internal/api/routes.go` | Create | `agent-lab/internal/api/routes.go` |
| `internal/api/api.go` | Create | `agent-lab/internal/api/api.go` |
| `cmd/server/http.go` | Create | `agent-lab/cmd/server/http.go` |
| `cmd/server/modules.go` | Create | `agent-lab/cmd/server/modules.go` |
| `cmd/server/main.go` | Replace | `agent-lab/cmd/server/main.go` + `server.go` |

## Validation

- `go build ./...` and `go vet ./...` pass
- `go mod tidy` produces no changes
- `docker compose up -d` starts PostgreSQL and Azurite
- `mise run dev` starts the server, connects to PostgreSQL and Blob Storage
- `GET /healthz` returns 200 with `{"status":"ok"}`
- `GET /readyz` returns 200 with `{"status":"ready"}` after startup complete
- SIGINT triggers graceful shutdown with ordered teardown log output
