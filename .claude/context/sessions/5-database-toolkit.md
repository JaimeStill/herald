# 5 - Database Toolkit

## Summary

Implemented PostgreSQL connection management, SQL query building, repository helpers, and pagination support as `pkg/` building blocks. All patterns adapted from agent-lab using `database/sql` with pgx as the registered driver via `pgx/v5/stdlib`.

## Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Database abstraction | `database/sql` + pgx stdlib driver | Idiomatic Go, standard interfaces, portable |
| Cold/hot start | `New()` calls `sql.Open()`, `Start()` registers lifecycle hooks | `sql.Open` is lazy — clean LCA separation |
| Row scanning | `ScanFunc[T]` | Explicit, type-safe, domain owns mapping |
| Repository interfaces | `Querier`, `Executor`, `Scanner` | Abstracts `*sql.DB`/`*sql.Tx` — helpers work in both contexts |
| Error mapping | `sql.ErrNoRows` + pgconn code 23505 | Domain-agnostic translation, callers provide error types |
| Query builder | ProjectionMap + Builder | Proven pattern from agent-lab, separates column defs from conditions |
| Docker volumes | Named volumes (Docker-managed) | Removes need for pre-existing host directories |

## Files Modified

### New Files
- `pkg/database/database.go` — System interface and lifecycle integration
- `pkg/database/errors.go` — `ErrNotReady` sentinel error
- `pkg/query/projection.go` — ProjectionMap for column mapping
- `pkg/query/builder.go` — Composable SQL builder with auto-numbered `$N` params
- `pkg/repository/repository.go` — Generic query helpers (`QueryOne`, `QueryMany`, `WithTx`, `ExecExpectOne`)
- `pkg/repository/errors.go` — `MapError` for domain-agnostic error translation
- `pkg/pagination/config.go` — Three-phase finalize config
- `pkg/pagination/pagination.go` — `PageRequest`, `PageResult`, `SortFields`

### Tests
- `tests/database/database_test.go` — System creation, pool params, ErrNotReady
- `tests/query/query_test.go` — ProjectionMap, ParseSortFields, all Builder methods
- `tests/repository/repository_test.go` — MapError (nil, ErrNoRows, PgError 23505, passthrough)
- `tests/pagination/pagination_test.go` — Config (defaults, env, validation, merge), PageRequest, PageResult, SortFields

### Deleted
- `pkg/query/doc.go` — replaced by projection.go and builder.go
- `pkg/repository/doc.go` — replaced by repository.go and errors.go
- `pkg/pagination/doc.go` — replaced by config.go and pagination.go

### Infrastructure
- `compose/azurite.yml` — fixed network (`external: true` → `driver: bridge`), switched to Docker-managed volumes
- `compose/postgres.yml` — switched to Docker-managed volumes

### Dependencies
- Added `github.com/jackc/pgx/v5` (pgx stdlib driver + pgconn error types)

## Patterns Established

- **Database lifecycle**: `New()` validates DSN + configures pool; `Start()` registers ping (startup) and close (shutdown) hooks
- **Projection-based queries**: ProjectionMap maps view names to qualified columns; Builder composes WHERE/ORDER BY/LIMIT
- **Generic repository helpers**: Type-safe via `ScanFunc[T]` generics, work with both `*sql.DB` and `*sql.Tx`
- **Domain-agnostic error mapping**: `MapError` translates DB errors using caller-provided domain error types
- **Pagination config**: Same three-phase finalize pattern as all other configs

## Validation Results

- `go build ./...` — pass
- `go vet ./...` — pass
- `go mod tidy` — clean
- `go test ./tests/...` — 12 packages, all pass
