# 7 - Infrastructure Assembly, API Module, and Server Entry Point

## Summary

Wired all foundational packages (lifecycle, database, storage, middleware, module/routing, config) into the Infrastructure assembly, API module shell, and server entry point. Herald is now a running Go web service with health/readiness probes, OpenAPI spec endpoint, Scalar API reference UI, and graceful shutdown following the Layered Composition Architecture from agent-lab.

## Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Server struct location | `cmd/server/server.go` | Matches agent-lab encapsulation; keeps `main.go` minimal |
| Logger creation | Inline in `infrastructure.New()` | No logging config needed yet; avoids unnecessary package |
| Domain struct | Empty placeholder | Domain systems added in Objectives #2 and #3 |
| Health response format | JSON `{"status":"ok"}` | Matches acceptance criteria; consistent with API responses |
| Scalar module | Wired in `cmd/server/modules.go` | Already implemented in `web/scalar/`; provides API documentation UI |
| Pagination config | Added to `APIConfig` | Required by API Runtime for domain system construction |

## Files Modified

- `internal/config/api.go` — added pagination config field, env mapping, finalize/merge wiring
- `config.toml` — added `[api.pagination]` section
- `internal/infrastructure/infrastructure.go` — new (replaced `doc.go` stub)
- `internal/api/api.go` — new (replaced `doc.go` stub)
- `internal/api/runtime.go` — new
- `internal/api/domain.go` — new
- `internal/api/routes.go` — new
- `cmd/server/main.go` — replaced stub with minimal entry point
- `cmd/server/server.go` — new (Server struct with lifecycle coordination)
- `cmd/server/http.go` — new (HTTP server wrapper with shutdown hook)
- `cmd/server/modules.go` — new (module assembly, health/readiness endpoints)
- `tests/infrastructure/infrastructure_test.go` — new
- `tests/api/api_test.go` — new
- `tests/config/config_test.go` — added pagination tests

## Patterns Established

- **Infrastructure assembly**: `New()` for cold start (no connections), `Start()` for hot start (lifecycle hooks)
- **API module composition**: Runtime → Domain → Module with middleware
- **Server lifecycle**: `NewServer()` → `Start()` → signal wait → `Shutdown()` in separate files (`server.go`, `http.go`, `modules.go`, `main.go`)
- **Health/readiness probes**: native router handlers, JSON responses, readiness gated by `lifecycle.Ready()`

## Validation Results

- `go build ./...` — pass
- `go vet ./...` — pass
- `go mod tidy` — no changes
- `go test ./tests/...` — 14 packages, all pass
- `mise run dev` — server starts, connects to PostgreSQL and Azurite, all subsystems ready
- `GET /healthz` — 200 `{"status":"ok"}`
- `GET /readyz` — 200 `{"status":"ready"}`
- SIGINT — graceful shutdown with ordered teardown
