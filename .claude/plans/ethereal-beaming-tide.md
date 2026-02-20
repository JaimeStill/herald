# Objective Planning: #2 — Database Schema and Migration Tooling

## Context

Objective #1 (Project Scaffolding, Configuration, and Service Skeleton) is 100% complete — all 4 sub-issues (#4–#7) are closed. Herald has a running Go web service with configuration, lifecycle, database connection pooling (`pkg/database/`), storage abstraction, and HTTP module infrastructure. The `cmd/migrate/main.go` stub exists but is empty.

Objective #2 establishes the database migration workflow and the initial PostgreSQL schema. This is a prerequisite for the document domain (Objective #3).

## Transition Closeout

- **Objective #1**: 4/4 sub-issues complete (100%) — close issue #1
- **No incomplete work** to carry forward or backlog
- Update `_project/phase.md` — mark Objective #1 status as Complete
- Delete `_project/objective.md` and recreate for Objective #2

## Sub-Issue Decomposition

Single sub-issue — the migration CLI, SQL files, and mise tasks are tightly coupled (the CLI embeds the SQL files via `go:embed`, and neither is testable independently).

### Sub-Issue: Migration CLI and Initial Schema

**Labels:** `feature`, `infrastructure`
**Milestone:** Phase 1

**Scope:**

1. **Add `golang-migrate` dependency** — `github.com/golang-migrate/migrate/v4` with `postgres` and `iofs` drivers
2. **Implement `cmd/migrate/main.go`** — standalone CLI following agent-lab's pattern (`~/code/agent-lab/cmd/migrate/main.go`):
   - `//go:embed migrations/*.sql` for self-contained binary
   - `-dsn` flag with `DATABASE_DSN` env var fallback (standalone — does not reuse config package)
   - Flags: `-up`, `-down`, `-steps N`, `-version`, `-force N`
   - Graceful `migrate.ErrNoChange` handling
   - Uses `iofs.New()` to wrap embedded FS as migration source
3. **SQL migration files** under `cmd/migrate/migrations/`:
   - `000001_initial_schema.{up,down}.sql` — `CREATE/DROP EXTENSION "uuid-ossp"`
   - `000002_documents.{up,down}.sql` — Documents table per Objective spec:
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
4. **mise tasks** in `.mise.toml`: `migrate:up`, `migrate:down`, `migrate:version` — each constructs DSN from local dev defaults and calls `go run ./cmd/migrate/`

**Acceptance Criteria:**
- `go build ./cmd/migrate/` succeeds
- `mise run migrate:up` applies both migrations successfully
- `mise run migrate:down` cleanly reverts all migrations
- `psql` confirms table structure, column types, and indexes

## Project Board Updates

- Create "Phase 1" milestone on herald repo (none exists yet)
- Add sub-issue to Herald project board (#7)
- Assign Phase: "Phase 1 - Service Foundation"
- Close Objective #1 after sub-issue creation and carry-forward handling

## Actions

1. Close Objective #1 (`gh issue close 1`)
2. Update `_project/phase.md` — mark Objective #1 as Complete
3. Create "Phase 1" milestone on herald repo
4. Create sub-issue with body, labels (`feature`, `infrastructure`), milestone
5. Assign `Task` issue type to sub-issue (if issue types are enabled)
6. Link sub-issue to Objective #2 as child via GraphQL
7. Add sub-issue to Herald project board, assign Phase
8. Recreate `_project/objective.md` for Objective #2
