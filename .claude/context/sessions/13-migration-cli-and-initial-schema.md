# 13 - Migration CLI and Initial Schema

## Summary

Implemented the standalone golang-migrate CLI (`cmd/migrate/`) with embedded SQL migrations and the initial PostgreSQL schema for the documents table. Established the migration workflow used throughout all phases via mise tasks.

## Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Single migration file | Combined UUID extension + documents table in `000001_initial_schema` | These are the initial schema and always applied together |
| Status column type | `TEXT` with `CHECK` constraint instead of PostgreSQL enum | Easier to modify via migration — no type recreation needed when adding values |
| Status values | `pending`, `review`, `complete` | Tracks document position in classification lifecycle, not operational concerns. Errors leave documents in `pending`; error handling and observability are separate concerns |
| Env var naming | `HERALD_DB_DSN` | Matches the `HERALD_DB_*` convention from `internal/config/` even though the CLI is standalone |
| Default DSN | `postgres://herald:herald@localhost:5432/herald?sslmode=disable` | Matches `config.toml` and Docker Compose defaults for zero-config local development |
| Force flag detection | `flag.Visit` instead of sentinel value | Avoids conflict between `-1` as default sentinel and as a valid argument |
| External identifiers | `external_id INTEGER` + `external_platform TEXT` with composite index | Links documents back to originating MSSQL Server databases by source instance |

## Files Modified

- `cmd/migrate/main.go` — full migration CLI replacing stub
- `cmd/migrate/migrations/000001_initial_schema.up.sql` — UUID extension, documents table, indexes
- `cmd/migrate/migrations/000001_initial_schema.down.sql` — reverse migration
- `.mise.toml` — `migrate:up`, `migrate:down`, `migrate:version` tasks + tombi schema directive
- `_project/README.md` — updated document status model and external identifier fields
- `_project/phase.md` — updated status transition reference
- `go.mod` / `go.sum` — golang-migrate dependency

## Patterns Established

- Migration CLI is standalone (`package main`) — does not import `internal/config/` or `pkg/database/`
- DSN precedence: `-dsn` flag → `HERALD_DB_DSN` env var → default DSN
- SQL migrations embedded via `//go:embed` and loaded with `iofs.New()`
- Status modeled as `TEXT` with `CHECK` constraint for maintainability

## Validation Results

- `go build ./cmd/migrate/` — passes
- `go vet ./...` — passes
- `go test ./tests/...` — all 14 existing test suites pass
- `mise run migrate:up` — applies migration successfully
- `mise run migrate:down` — reverts migration cleanly
- PostgreSQL confirms table structure, column types, indexes, and CHECK constraint
