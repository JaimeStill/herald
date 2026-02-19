# Phase Planning: Phase 1 — Service Foundation (v0.1.0)

## Context

Herald has a complete concept document (`_project/README.md`) and bootstrapped GitHub infrastructure (project board #7, 4 milestones, 8 labels) but zero source code. Phase 1 establishes the Go service foundation — from `go.mod` to a running server that accepts document uploads and manages them through Azure Blob Storage and PostgreSQL.

The implementation draws heavily from proven patterns in agent-lab (infrastructure, config, lifecycle, domains, HTTP) and classify-docs (document handling). The project uses **mise** for task running.

## Phase 1 Scope (from concept)

> Go project scaffolding, Azure PostgreSQL schema/migrations, Azure Blob Storage integration, configuration system, module/routing infrastructure, document domain (upload single + batch, registration, metadata), storage abstraction, lifecycle coordination

## Objectives

### Objective 1: Project Scaffolding, Configuration, and Service Skeleton

**What**: Take Herald from zero to a running Go web service with health/readiness probes and graceful shutdown.

**Scope**:
- Go module init (`github.com/JaimeStill/herald`)
- Full directory structure per concept (`cmd/`, `internal/`, `pkg/`)
- `.mise.toml` with tasks: `dev`, `build`, `run`, `test`, `vet`
- Configuration system (`internal/config/`): TOML base + env overlay + env var overrides, three-phase finalization (adapted from agent-lab `internal/config/config.go`)
- `pkg/lifecycle` — Coordinator with startup/shutdown hooks, context cancellation (adapted from agent-lab `pkg/lifecycle/`)
- `pkg/database` — System interface, pgx connection management (adapted from agent-lab `pkg/database/`)
- `pkg/storage` — System interface + Azure Blob Storage implementation via azblob SDK (agent-lab's interface, new implementation)
- `pkg/middleware` — CORS, logging (adapted from agent-lab `pkg/middleware/`)
- `pkg/module` — Module + Router for prefix-based routing (adapted from agent-lab `pkg/module/`)
- `pkg/handlers` — RespondJSON, RespondError (adapted from agent-lab `pkg/handlers/`)
- `pkg/routes` — Route/Group registration types, no OpenAPI (simplified from agent-lab `pkg/routes/`)
- `pkg/query` — SQL query builder with projections, conditions, pagination (adapted from agent-lab `pkg/query/`)
- `pkg/repository` — QueryOne, QueryMany, WithTx, ExecExpectOne (adapted from agent-lab `pkg/repository/`)
- `pkg/pagination` — PageRequest, PageResult[T], SortFields (adapted from agent-lab `pkg/pagination/`)
- `internal/infrastructure` — Infrastructure struct (Lifecycle, Logger, Database, Storage), assembly (adapted from agent-lab `internal/infrastructure/`)
- `internal/api` — Runtime + empty Domain, API module with middleware (adapted from agent-lab `internal/api/`)
- `cmd/server/` — Full cold start -> hot start -> shutdown lifecycle (adapted from agent-lab `cmd/server/`)
- `config.toml` with all sections
- Docker Compose for local PostgreSQL + Azurite

**Out of scope**: Migrations, domain logic, web client, auth

**Dependencies**: None (first objective)

**Verification**:
- `go build ./...` and `go vet ./...` pass
- `go test ./...` passes (unit tests for config, lifecycle, query builder, pagination, repository helpers)
- `mise run dev` starts server, connects to PostgreSQL and Blob Storage
- `GET /healthz` returns 200, `GET /readyz` returns 200 after startup
- SIGINT triggers graceful shutdown

---

### Objective 2: Database Schema and Migration Tooling

**What**: Establish the migration workflow and document table schema.

**Scope**:
- `cmd/migrate/` — golang-migrate CLI with embedded SQL, flags: `-up`, `-down`, `-steps`, `-version`, `-force` (adapted from agent-lab `cmd/migrate/`)
- `cmd/migrate/migrations/000001_initial_schema.{up,down}.sql` — UUID extension
- `cmd/migrate/migrations/000002_documents.{up,down}.sql` — Documents table:
  ```sql
  CREATE TABLE documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    filename TEXT NOT NULL,
    content_type TEXT NOT NULL,
    size_bytes BIGINT NOT NULL,
    page_count INTEGER,
    storage_key TEXT NOT NULL UNIQUE,
    status TEXT NOT NULL DEFAULT 'pending',
    uploaded_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
  );
  ```
  Plus indexes on `status`, `filename`, `uploaded_at`
- mise tasks: `migrate:up`, `migrate:down`, `migrate:version`

**Out of scope**: Classifications table (Phase 2), prompt_modifications table (Phase 2)

**Dependencies**: Objective 1 (Go module, database config, pkg/database)

**Verification**:
- `go build ./cmd/migrate/` succeeds
- `mise run migrate:up` applies both migrations
- `mise run migrate:down` cleanly reverts
- `psql` confirms table structure and indexes

---

### Objective 3: Document Domain

**What**: Deliver the complete document lifecycle — upload, registration, metadata, storage, and CRUD queries.

**Scope**:
- `internal/documents/` — Full domain following agent-lab's System/Repository/Handler pattern:
  - **System interface**: List, Find, Create, CreateBatch, Delete
  - **Repository**: pgx + query builder, blob storage coordination, DB+blob atomicity
  - **Handler**: Routes at `/documents` with endpoints:
    - `GET /documents` — Paginated list with filters
    - `GET /documents/{id}` — Find by ID
    - `POST /documents` — Single upload (multipart)
    - `POST /documents/batch` — Batch upload (multipart, multiple files)
    - `POST /documents/search` — Search with JSON body
    - `DELETE /documents/{id}` — Delete (DB + blob)
  - **Types**: Document struct, CreateCommand, Filters, domain errors
  - **Mapping**: ProjectionMap, scanDocument, FiltersFromQuery
- PDF page count extraction via pdfcpu on upload
- Storage key: `documents/{id}/{filename}.pdf`
- Wire into `internal/api` Domain struct and route registration

**Out of scope**: File content download endpoint, classification status transitions (Phase 2), external system identifiers (deferred)

**Dependencies**: Objective 1 (service skeleton, all pkg/ packages) + Objective 2 (documents schema)

**Verification**:
- `go test ./...` passes (domain unit tests in `tests/` using black-box pattern)
- Upload single PDF → 201 with document JSON (ID, storage key, page count, status "pending")
- Upload batch → 201 with array of documents
- List with pagination and filters → paginated results
- Search with JSON body → filtered results
- Find by ID → single document
- Delete → 204, blob removed from storage
- Verify blob exists in storage at correct key after upload

## Dependency Graph

```
Objective 1: Project Scaffolding, Configuration, and Service Skeleton
    │
    v
Objective 2: Database Schema and Migration Tooling
    │
    v
Objective 3: Document Domain
```

## Key Reference Files

| Pattern | Source (agent-lab) |
|---------|-------------------|
| Server composition | `cmd/server/server.go` |
| Configuration system | `internal/config/config.go` |
| Infrastructure assembly | `internal/infrastructure/infrastructure.go` |
| Domain pattern | `internal/documents/` |
| Database toolkit | `pkg/database/`, `pkg/query/`, `pkg/repository/` |
| Storage interface | `pkg/storage/storage.go` |
| HTTP infrastructure | `pkg/module/`, `pkg/middleware/`, `pkg/handlers/`, `pkg/routes/` |
| Pagination | `pkg/pagination/` |
| Lifecycle | `pkg/lifecycle/` |
| Migration CLI | `cmd/migrate/` |
