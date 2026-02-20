# Objective: Database Schema and Migration Tooling

**Parent Issue:** [#2](https://github.com/JaimeStill/herald/issues/2)
**Phase:** Phase 1 — Service Foundation (v0.1.0)
**Repository:** herald

## Scope

Establish the database migration workflow and the initial PostgreSQL schema for the documents table. This creates the migration CLI and the schema foundation that the document domain depends on, and establishes the migration workflow used throughout all phases.

## Sub-Issues

| # | Title | Labels | Status | Dependencies |
|---|-------|--------|--------|--------------|
| [#13](https://github.com/JaimeStill/herald/issues/13) | Migration CLI and Initial Schema | `feature`, `infrastructure` | Open | — |

## Architecture Decisions

### Standalone Migration CLI

The migration CLI (`cmd/migrate/`) is standalone — it does not reuse the config package. Database connection is provided via `-dsn` flag or `DATABASE_DSN` env var. This follows agent-lab's proven pattern and keeps the migration tool simple and independently deployable.

### Embedded Migrations

SQL migration files are embedded in the binary via `//go:embed migrations/*.sql` and loaded with `iofs.New()`. This makes the migration binary self-contained with no external file dependencies.

### Reference Pattern

agent-lab `cmd/migrate/` — same golang-migrate wrapper with embedded SQL, `-dsn`/`DATABASE_DSN` connection, and standard flags (`-up`, `-down`, `-steps`, `-version`, `-force`).

## Verification

- `go build ./cmd/migrate/` succeeds
- `mise run migrate:up` applies both migrations successfully
- `mise run migrate:down` cleanly reverts all migrations
- `psql` confirms table structure, column types, and indexes
