# Issue #4 — Project Scaffolding, Configuration, Lifecycle, and HTTP Infrastructure

## Context

Herald has zero Go source code. This task establishes the Go module, full directory structure, build tooling, local development environment, and implements the foundational packages: configuration system (TOML + env overlays + env var overrides), lifecycle coordinator (startup/shutdown hooks), HTTP infrastructure (middleware, module routing, response helpers, route registration), and OpenAPI specification infrastructure. All patterns adapted from agent-lab (`~/code/agent-lab`), with Herald-specific adjustments (Azure Blob Storage config, `HERALD_` env prefix, CDN-based Scalar for initial development).

## Pre-Implementation: CLAUDE.md

Initialize `.claude/CLAUDE.md` with project conventions before writing the implementation guide. Key convention: **OpenAPI schema maintenance is an AI responsibility** — once the implementation plan is accepted, the AI completes anticipated OpenAPI schema adjustments before transitioning control to the developer.

## Implementation Steps

### Step 1: Project Scaffolding

- `go mod init github.com/JaimeStill/herald` + `go-toml` dependency
- `.gitignore` — Go binaries, test output, coverage, IDE files, config overlays (`config.*.toml`), build artifacts (`bin/`)
- `doc.go` stubs for future packages: `internal/api/`, `internal/infrastructure/`, `internal/documents/`, `internal/classifications/`, `internal/prompts/`, `workflow/`, `pkg/pagination/`, `pkg/query/`, `pkg/repository/`, `pkg/web/`

### Step 2: Build Tooling

**`.mise.toml`** — tasks: `dev` (go run), `build` (go build -o bin/server), `test` (go test), `vet` (go vet)

**Docker Compose** — following agent-lab's compose-include pattern:
- `docker-compose.yml` — includes `compose/postgres.yml` + `compose/azurite.yml`
- `compose/postgres.yml` — PostgreSQL 17, container `herald-postgres`, db/user/password default `herald`, port 5432, healthcheck, named network `herald`
- `compose/azurite.yml` — Azure Storage emulator, container `herald-azurite`, blob port 10000, uses `herald` network (external)

**`config.toml`** — base configuration skeleton with sections: server, database, storage, api (cors + openapi)

### Step 3: Package Config Types

Config/Env types in their respective packages (following agent-lab's "define where implemented" pattern). Each type implements `Finalize(env)` (loadDefaults → loadEnv → validate) and `Merge(overlay)`.

| File | Types | Key Fields |
|------|-------|------------|
| `pkg/database/config.go` | `Config`, `Env` | host, port, name, user, password, ssl_mode, pool settings; `Dsn()` method |
| `pkg/storage/config.go` | `Config`, `Env` | container_name, connection_string |
| `pkg/middleware/config.go` | `CORSConfig`, `CORSEnv` | enabled, origins, methods, headers, credentials, max_age |
| `pkg/openapi/config.go` | `Config`, `ConfigEnv` | title, description |

### Step 4: Configuration System (`internal/config/`)

Root config with three-phase finalization, adapted from agent-lab `internal/config/`.

| File | Responsibility |
|------|---------------|
| `config.go` | Root `Config` struct (embeds ServerConfig, database.Config, storage.Config, APIConfig), `Load()` (base → overlay → finalize), `Merge()`, env overlay via `HERALD_ENV` |
| `server.go` | `ServerConfig` — host, port, read/write/shutdown timeouts; `Addr()`, duration parsers, Finalize/Merge |
| `api.go` | `APIConfig` — base_path, CORS, OpenAPI; delegates CORSConfig.Finalize with CORSEnv and openapi.Config.Finalize with ConfigEnv |

**Env var convention**: `HERALD_` prefix — `HERALD_ENV`, `HERALD_SHUTDOWN_TIMEOUT`, `HERALD_VERSION`, `HERALD_SERVER_PORT`, `HERALD_DB_HOST`, `HERALD_STORAGE_CONTAINER_NAME`, `HERALD_CORS_ENABLED`, `HERALD_OPENAPI_TITLE`, etc.

**Env struct injection**: `internal/config/` instantiates env structs with concrete var names and passes them to sub-package `Finalize()` methods.

### Step 5: Lifecycle Coordinator (`pkg/lifecycle/`)

Direct adaptation of `~/code/agent-lab/pkg/lifecycle/lifecycle.go`:
- `Coordinator` with `OnStartup(fn)`, `OnShutdown(fn)`, `WaitForStartup()`, `Shutdown(timeout)`
- Uses `sync.WaitGroup.Go()` for hook execution
- Shutdown hooks block on `<-lc.Context().Done()` before executing cleanup
- `Ready()` returns true only after all startup hooks complete

### Step 6: OpenAPI Infrastructure (`pkg/openapi/`)

Adapted from `~/code/agent-lab/pkg/openapi/`:

| File | Responsibility |
|------|---------------|
| `types.go` | Core OpenAPI 3.1 types: Spec, Info, Server, PathItem, Operation, Parameter, RequestBody, Response, MediaType, Schema, Components + helper constructors (SchemaRef, ResponseRef, RequestBodyJSON, ResponseJSON, PathParam, QueryParam) |
| `spec.go` | `NewSpec(title, version)`, `AddServer`, `SetDescription`, `ServeSpec(bytes)` handler |
| `components.go` | `NewComponents()` with shared error responses (BadRequest, NotFound, Conflict), `AddSchemas`, `AddResponses` |
| `json.go` | `MarshalJSON(spec)` and `WriteJSON(spec, filename)` serialization |
| `config.go` | `Config` / `ConfigEnv` with title, description; Finalize/Merge pattern |

### Step 7: HTTP Infrastructure

**`pkg/handlers/handlers.go`** — `RespondJSON(w, status, data)`, `RespondError(w, logger, status, err)` — stateless response utilities.

**`pkg/routes/`** — with OpenAPI integration (adapted from agent-lab):
- `route.go` — `Route{Method, Pattern, Handler, OpenAPI}` where OpenAPI is `*openapi.Operation` (nil means undocumented)
- `group.go` — `Group{Prefix, Tags, Description, Routes, Children, Schemas}`, `Register(mux, basePath, spec, groups...)` walks tree registering handlers and populating the OpenAPI spec

**`pkg/middleware/`** — adapted from agent-lab:
- `middleware.go` — `System` interface (`Use`, `Apply`), reverse-order application
- `cors.go` — `CORS(cfg)` middleware function, origin allowlist, preflight handling
- `logger.go` — `Logger(logger)` post-request logging with method, URI, addr, duration

**`pkg/module/`** — adapted from agent-lab:
- `module.go` — `Module` with prefix validation, prefix stripping, middleware application
- `router.go` — `Router` with module dispatch by first path segment + native handler fallback

### Step 8: Scalar Module (`web/scalar/`)

CDN-based Scalar for initial development (no Bun/Vite build step required). Switch to embedded assets when the full web client build pipeline is established in Phase 3.

- `scalar.go` — `NewModule(basePath)` creates a module serving `index.html` as a Go template with CDN-loaded Scalar JS/CSS
- `index.html` — HTML template with `{{ .BasePath }}` injection, loads `@scalar/api-reference` from CDN, points spec URL at `/api/openapi.json`

### Step 9: Walking Skeleton (`cmd/server/main.go`)

Minimal entry point proving compilation: loads config, logs startup info, waits for SIGINT/SIGTERM, logs shutdown. Exercises the config package at runtime. Full server wiring (infrastructure assembly, module mounting, HTTP listening, OpenAPI spec generation, Scalar mounting) deferred to issue #7.

## Adaptations from agent-lab

| Area | agent-lab | Herald |
|------|-----------|--------|
| Storage config | Filesystem (base_path, max_upload_size) | Azure Blob Storage (container_name, connection_string) |
| API config | CORS + OpenAPI + Pagination | CORS + OpenAPI (pagination added in #7) |
| Server entry | Full infrastructure assembly | Walking skeleton (full assembly in #7) |
| Logging | Dedicated `pkg/logging/` package | `slog.New()` directly (defer logging package if needed) |
| Scalar assets | Embedded via `go:embed` (Vite build) | CDN-based initially (embedded in Phase 3) |

## Files Created

**Scaffolding** (~12 doc.go stubs + go.mod + .gitignore)
**Build tooling** (5 files: .mise.toml, docker-compose.yml, compose/postgres.yml, compose/azurite.yml, config.toml)
**Config types** (4 files: pkg/database/config.go, pkg/storage/config.go, pkg/middleware/config.go, pkg/openapi/config.go)
**Config system** (3 files: internal/config/config.go, server.go, api.go)
**Lifecycle** (1 file: pkg/lifecycle/lifecycle.go)
**OpenAPI** (4 files: pkg/openapi/types.go, spec.go, components.go, json.go)
**HTTP infra** (8 files: handlers.go, route.go, group.go, middleware.go, cors.go, logger.go, module.go, router.go)
**Scalar** (2 files: web/scalar/scalar.go, web/scalar/index.html)
**Skeleton** (1 file: cmd/server/main.go)
**Project CLAUDE.md** (1 file: .claude/CLAUDE.md)

**Total: ~41 new files**

## Verification

- `go build ./...` and `go vet ./...` pass
- `mise run build` produces `bin/server`
- `docker compose up -d` brings up PostgreSQL (5432) and Azurite (10000)
- Unit tests (authored by AI in Phase 5) for config, lifecycle, middleware, routes, handlers, openapi
