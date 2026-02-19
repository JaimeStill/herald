# Objective: Project Scaffolding, Configuration, and Service Skeleton

**Parent Issue:** [#1](https://github.com/JaimeStill/herald/issues/1)
**Phase:** Phase 1 — Service Foundation (v0.1.0)
**Repository:** herald

## Scope

Take Herald from zero source code to a running Go web service with health/readiness probes and graceful shutdown. Establishes the configuration system, lifecycle coordinator, infrastructure layer, HTTP module/routing infrastructure, database toolkit, storage abstraction, and the API module shell.

## Sub-Issues

| # | Title | Labels | Status | Dependencies |
|---|-------|--------|--------|--------------|
| [#4](https://github.com/JaimeStill/herald/issues/4) | Project Scaffolding, Configuration, Lifecycle, and HTTP Infrastructure | `infrastructure`, `feature` | Open | — |
| [#5](https://github.com/JaimeStill/herald/issues/5) | Database Toolkit | `feature` | Open | #4 |
| [#6](https://github.com/JaimeStill/herald/issues/6) | Storage Abstraction | `feature` | Open | #4 |
| [#7](https://github.com/JaimeStill/herald/issues/7) | Infrastructure Assembly, API Module, and Server Entry Point | `feature` | Open | #4, #5, #6 |

## Dependency Graph

```
#4: Scaffolding + Config + Lifecycle + HTTP
    ├── #5: Database Toolkit
    └── #6: Storage Abstraction
        └── #7: Infrastructure Assembly + API Module + Server Entry Point
```

Sub-issues #5 and #6 can proceed in parallel after #4. Sub-issue #7 depends on all three.

## Architecture Decisions

### Layered Composition Architecture (LCA)

Herald follows the cold start / hot start / shutdown pattern from agent-lab:

1. **Cold start** — configuration loading, subsystem creation (no connections)
2. **Hot start** — database connect, storage initialize, HTTP listen
3. **Shutdown** — reverse-order teardown within deadline

### Reference Patterns

All patterns are adapted from agent-lab. Each sub-issue body identifies the specific source files.

| Package | Source Pattern |
|---------|--------------|
| `internal/config/` | `agent-lab/internal/config/config.go` |
| `pkg/lifecycle/` | `agent-lab/pkg/lifecycle/` |
| `pkg/middleware/` | `agent-lab/pkg/middleware/` |
| `pkg/module/` | `agent-lab/pkg/module/` |
| `pkg/handlers/` | `agent-lab/pkg/handlers/` |
| `pkg/routes/` | `agent-lab/pkg/routes/` |
| `pkg/database/` | `agent-lab/pkg/database/` |
| `pkg/query/` | `agent-lab/pkg/query/` |
| `pkg/repository/` | `agent-lab/pkg/repository/` |
| `pkg/pagination/` | `agent-lab/pkg/pagination/` |
| `pkg/storage/` | `agent-lab/pkg/storage/storage.go` |
| `internal/infrastructure/` | `agent-lab/internal/infrastructure/infrastructure.go` |
| `internal/api/` | `agent-lab/internal/api/` |
| `cmd/server/` | `agent-lab/cmd/server/server.go` |

## Verification

- `go build ./...` and `go vet ./...` pass
- `go test ./...` passes with all unit tests
- `mise run dev` starts the server, connects to PostgreSQL and Blob Storage
- `GET /healthz` returns 200, `GET /readyz` returns 200 after startup
- SIGINT triggers graceful shutdown
