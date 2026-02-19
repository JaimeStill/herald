# Objective Planning: Issue #1 — Project Scaffolding, Configuration, and Service Skeleton

## Context

Issue #1 is the first Objective in Phase 1 (Service Foundation). It takes Herald from zero source code to a running Go web service with health/readiness probes and graceful shutdown. This objective establishes every foundational package that Objectives #2 and #3 depend on.

No previous objective exists — this is a clean start. The codebase currently has no Go source code.

## Sub-Issue Decomposition

Four sub-issues with a linear dependency chain:

```
Sub-Issue 1: Scaffolding + Config + Lifecycle + HTTP
    └── Sub-Issue 2: Database Toolkit
    └── Sub-Issue 3: Storage Abstraction
        └── Sub-Issue 4: Infrastructure Assembly + API Module + Server Entry Point
```

Sub-Issues 2 and 3 can proceed in parallel after Sub-Issue 1. Sub-Issue 4 depends on all three.

### Sub-Issue 1: Project Scaffolding, Configuration, Lifecycle, and HTTP Infrastructure
- **Labels**: `infrastructure`, `feature`
- **Scope**:
  - `go mod init github.com/JaimeStill/herald`, full directory structure
  - `.mise.toml` with tasks: dev, build, run, test, vet
  - Docker Compose for PostgreSQL + Azurite
  - `config.toml` skeleton, `.gitignore`
  - `internal/config/` — TOML base + env overlay + env var overrides, three-phase finalization
  - `pkg/lifecycle/` — Coordinator with startup/shutdown hooks, context cancellation, `WaitForStartup()`, `Shutdown(timeout)`
  - `pkg/middleware/` — CORS, request logging
  - `pkg/module/` — Module + Router for prefix-based routing
  - `pkg/handlers/` — RespondJSON, RespondError
  - `pkg/routes/` — Route/Group registration types
- **Acceptance**:
  - `go build ./...` and `go vet ./...` pass
  - `mise run build` works, Docker Compose brings up PostgreSQL and Azurite
  - Unit tests pass for config (TOML parsing, env overlay, env var overrides), lifecycle (hook ordering, shutdown timeout, context cancellation), middleware, route registration, JSON response helpers

### Sub-Issue 2: Database Toolkit
- **Labels**: `feature`
- **Scope**:
  - `pkg/database/` — System interface, pgx pool management, health check
  - `pkg/query/` — SQL query builder with projections, conditions, sorting
  - `pkg/repository/` — QueryOne, QueryMany, WithTx, ExecExpectOne
  - `pkg/pagination/` — PageRequest, PageResult[T], SortFields
- **Depends on**: Sub-Issue 1
- **Acceptance**: Unit tests pass for query builder, pagination math, repository helpers (mock-based)

### Sub-Issue 3: Storage Abstraction
- **Labels**: `feature`
- **Scope**:
  - `pkg/storage/` — System interface (Upload, Download, Delete, Exists)
  - Azure Blob Storage implementation using azblob SDK
  - Container initialization on startup
- **Depends on**: Sub-Issue 1
- **Acceptance**: Unit tests pass for interface contract; integration test with Azurite verifies upload/download/delete

### Sub-Issue 4: Infrastructure Assembly, API Module, and Server Entry Point
- **Labels**: `feature`
- **Scope**:
  - `internal/infrastructure/` — Infrastructure struct (Lifecycle, Logger, Database, Storage), `New()` constructor, `Start()`/`Shutdown()` methods
  - `internal/api/` — Runtime embedding Infrastructure + pagination config, empty Domain struct, API module with middleware
  - `cmd/server/` — Full cold start → hot start → shutdown lifecycle, signal handling (SIGINT/SIGTERM)
  - Health probe (`GET /healthz`) and readiness probe (`GET /readyz`)
- **Depends on**: Sub-Issues 1, 2, 3
- **Acceptance**:
  - `mise run dev` starts server, connects to PostgreSQL and Azurite
  - `GET /healthz` returns 200, `GET /readyz` returns 200 after startup
  - SIGINT triggers graceful shutdown

## Project Board Operations

For each sub-issue:
1. Create issue on `JaimeStill/herald` with body, labels, and milestone "v0.1.0 - Service Foundation"
2. Link as sub-issue to parent #1 via GraphQL
3. Add to Herald project board (#7)
4. Assign Phase "Phase 1 - Service Foundation"

## `_project/objective.md`

Create with:
- Objective title and issue #1 reference
- Phase reference (Phase 1 — Service Foundation, v0.1.0)
- Sub-issues table with status tracking
- Dependency graph between sub-issues
- Reference patterns from agent-lab for each sub-issue

## Verification

- All 4 sub-issues visible as sub-issues of #1 on GitHub
- All 4 sub-issues on Herald project board in Phase 1
- All 4 sub-issues have milestone "v0.1.0 - Service Foundation"
- `_project/objective.md` committed with complete sub-issue table
