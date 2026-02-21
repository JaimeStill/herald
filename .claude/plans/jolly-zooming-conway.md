# Plan: Issue #13 — Migration CLI and Initial Schema

## Context

Herald needs a database migration workflow before the document domain (Objective #3) can be built. The `cmd/migrate/main.go` exists as a stub. This task implements the standalone golang-migrate CLI with embedded SQL files and two initial migrations (UUID extension + documents table), following agent-lab's proven pattern exactly.

## Architecture Approach

The migration CLI is **standalone** — it does not import `internal/config/` or `pkg/database/`. Connection is via `-dsn` flag or `DATABASE_DSN` env var. Migrations are embedded in the binary via `//go:embed`. This matches agent-lab's pattern and keeps the tool independently deployable.

The default DSN (`postgres://herald:herald@localhost:5432/herald?sslmode=disable`) matches `config.toml` and Docker Compose defaults so `mise run migrate:up` works out of the box for local development.

## Implementation

### Step 1: Add golang-migrate dependency

```bash
go get github.com/golang-migrate/migrate/v4
go get github.com/golang-migrate/migrate/v4/database/postgres
go get github.com/golang-migrate/migrate/v4/source/iofs
```

### Step 2: Implement `cmd/migrate/main.go`

Replace the stub with the full CLI. Follows agent-lab's pattern exactly:

- `//go:embed migrations/*.sql` for embedded migrations
- Flag set: `-dsn`, `-up`, `-down`, `-steps N`, `-version`, `-force N`
- DSN precedence: `-dsn` flag → `DATABASE_DSN` env var → default DSN → error (never reached since default exists)
- `iofs.New(migrations, "migrations")` for source
- `migrate.NewWithSourceInstance("iofs", source, dsn)` for migrator
- Graceful `migrate.ErrNoChange` handling on `-up` and `-down`

**Key files:** `cmd/migrate/main.go`

### Step 3: Create SQL migration files

Single migration combining the UUID extension and documents table — these are the initial schema and always applied together.

**`cmd/migrate/migrations/000001_initial_schema.up.sql`**
- `CREATE EXTENSION IF NOT EXISTS "uuid-ossp"` (provides `uuid_generate_v4()`; `gen_random_uuid()` is built-in since PostgreSQL 13, but the extension is included for completeness and compatibility)
- Documents table per issue spec:
  - `id UUID PRIMARY KEY DEFAULT gen_random_uuid()`
  - `filename TEXT NOT NULL`
  - `content_type TEXT NOT NULL`
  - `size_bytes BIGINT NOT NULL`
  - `page_count INTEGER`
  - `storage_key TEXT NOT NULL UNIQUE`
  - `status TEXT NOT NULL DEFAULT 'pending'`
  - `uploaded_at TIMESTAMPTZ DEFAULT NOW()`
  - `updated_at TIMESTAMPTZ DEFAULT NOW()`
- Indexes on `status`, `filename`, `uploaded_at`

**`cmd/migrate/migrations/000001_initial_schema.down.sql`**
- `DROP TABLE IF EXISTS documents`
- `DROP EXTENSION IF EXISTS "uuid-ossp"`

### Step 4: Add mise tasks

Add to `.mise.toml`:
- `migrate:up` — `go run ./cmd/migrate -up`
- `migrate:down` — `go run ./cmd/migrate -down`
- `migrate:version` — `go run ./cmd/migrate -version`

### Step 5: Run `go mod tidy`

Ensure module dependencies are clean after adding golang-migrate.

## Validation Criteria

- [ ] `go build ./cmd/migrate/` succeeds
- [ ] `go vet ./...` passes
- [ ] `mise run migrate:up` applies both migrations against local PostgreSQL
- [ ] `mise run migrate:down` cleanly reverts all migrations
- [ ] `psql` confirms table structure, column types, and indexes
- [ ] `go mod tidy` produces no changes
