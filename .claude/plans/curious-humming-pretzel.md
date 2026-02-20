# Issue #5 — Database Toolkit

## Context

Herald needs PostgreSQL connection management, SQL query building, repository helpers, and pagination support. These are the `pkg/` building blocks consumed by domain systems (documents, classifications, prompts) in later objectives. All patterns are adapted directly from agent-lab, using `database/sql` as the standard Go database abstraction with pgx as the registered driver via `pgx/v5/stdlib`.

## Key Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Database abstraction | `database/sql` + pgx stdlib driver | Idiomatic Go. Standard interfaces, portable, ecosystem-compatible. Same pattern as agent-lab. |
| Cold/hot start | `New()` calls `sql.Open()` (validates DSN, no connection); `Start()` registers ping + close hooks | `sql.Open` is lazy — no actual connection until first use or ping. Clean LCA separation. |
| Row scanning | `ScanFunc[T]` (domain packages define scan functions) | Explicit, type-safe, no struct tag magic. Domain owns its mapping. |
| Repository interfaces | `Querier`, `Executor`, `Scanner` | Abstracts `*sql.DB`, `*sql.Tx`, `*sql.Conn` — helpers work in both normal and transaction contexts. |
| Error mapping | `sql.ErrNoRows` + `pgconn.PgError` code 23505 | Domain-agnostic translation. Callers provide their own error types. |
| Query builder | ProjectionMap + Builder (from agent-lab) | Proven pattern; separates reusable column definitions from per-query conditions. |

## Dependency

Add `github.com/jackc/pgx/v5` — used via `pgx/v5/stdlib` driver registration and `pgconn.PgError` for error mapping.

## Files

### `pkg/database/database.go` — NEW

System interface and `database/sql` lifecycle integration. Adapted from agent-lab's `pkg/database/database.go`.

```go
type System interface {
    Connection() *sql.DB
    Start(lc *lifecycle.Coordinator) error
}
```

- `New(cfg *Config, logger *slog.Logger) (System, error)` — calls `sql.Open("pgx", cfg.Dsn())`, sets pool params (`SetMaxOpenConns`, `SetMaxIdleConns`, `SetConnMaxLifetime`)
- `Start(lc)` — registers startup hook (ping with ConnTimeout), registers shutdown hook (blocks on `<-lc.Context().Done()` then `conn.Close()`)
- `Connection()` — returns `*sql.DB`
- Driver import: `_ "github.com/jackc/pgx/v5/stdlib"`

### `pkg/database/errors.go` — NEW

```go
var ErrNotReady = errors.New("database not ready")
```

### `pkg/query/projection.go` — NEW (replaces doc.go)

Adapted directly from agent-lab (`pkg/query/projection.go`). Maps view property names to qualified `alias.column` references.

- `ProjectionMap` struct with schema, table, alias, column map
- `NewProjectionMap(schema, table, alias)` constructor
- `Project(column, viewName)` — add column mapping
- `Table()` — returns `schema.table alias`
- `Column(viewName)` — qualified column lookup
- `Columns()` — comma-separated column list
- `ColumnList()` — slice of qualified columns
- `Alias()` — table alias

### `pkg/query/builder.go` — NEW

Adapted directly from agent-lab (`pkg/query/builder.go`). Composable SQL query builder with auto-numbered `$N` parameters.

- `SortField{Field, Descending}` struct
- `ParseSortFields(string) []SortField` — parses `"name,-createdAt"` format
- `NewBuilder(projection, defaultSort...)` constructor
- WHERE methods: `WhereEquals`, `WhereContains` (ILIKE), `WhereIn`, `WhereNullable`, `WhereSearch` (multi-field OR)
- All WHERE methods are nil-safe (no-op on nil/empty input)
- `OrderByFields([]SortField)` — explicit sort override
- Build methods: `Build()`, `BuildCount()`, `BuildPage(page, pageSize)`, `BuildSingle(idField, id)`, `BuildSingleOrNull()`
- Helper: `isNil()` — reflection-based nil detection for interface values

### `pkg/repository/repository.go` — NEW (replaces doc.go)

Adapted from agent-lab (`pkg/repository/repository.go`). Generic query helpers using `database/sql` interfaces.

- `Querier` interface — `QueryContext`, `QueryRowContext` (implemented by `*sql.DB`, `*sql.Tx`)
- `Executor` interface — `ExecContext` (implemented by `*sql.DB`, `*sql.Tx`)
- `Scanner` interface — `Scan(dest ...any) error` (implemented by `*sql.Row`, `*sql.Rows`)
- `ScanFunc[T any]` — `func(Scanner) (T, error)` for domain scan functions
- `QueryOne[T](ctx, q Querier, query string, args []any, scan ScanFunc[T]) (T, error)` — single row
- `QueryMany[T](ctx, q Querier, query string, args []any, scan ScanFunc[T]) ([]T, error)` — multi row
- `WithTx[T](ctx, db *sql.DB, fn func(*sql.Tx) (T, error)) (T, error)` — transaction wrapper with auto rollback
- `ExecExpectOne(ctx, e Executor, query string, args ...any) error` — exec expecting one affected row

### `pkg/repository/errors.go` — NEW

Adapted from agent-lab (`pkg/repository/errors.go`).

- `MapError(err, notFoundErr, duplicateErr error) error` — maps `sql.ErrNoRows` → notFoundErr, pgconn code `23505` → duplicateErr
- Keeps repository domain-agnostic; callers provide their domain error types

### `pkg/pagination/config.go` — NEW (replaces doc.go)

Adapted from agent-lab (`pkg/pagination/config.go`). Three-phase finalize pattern.

- `Config{DefaultPageSize, MaxPageSize}` with `Finalize(*ConfigEnv)` and `Merge(*Config)`
- `ConfigEnv{DefaultPageSize, MaxPageSize}` for env var name injection
- Defaults: DefaultPageSize=20, MaxPageSize=100
- Validation: both positive, default <= max

### `pkg/pagination/pagination.go` — NEW

Adapted from agent-lab (`pkg/pagination/pagination.go`).

- `SortFields` — wraps `[]query.SortField` with flexible JSON unmarshal (string `"name,-created_at"` or array format)
- `PageRequest{Page, PageSize, Search *string, Sort SortFields}` with `Normalize(Config)` and `Offset()`
- `PageRequestFromQuery(url.Values, Config) PageRequest` — parses `page`, `page_size`, `search`, `sort` query params
- `PageResult[T]{Data []T, Total, Page, PageSize, TotalPages}` with `NewPageResult[T]()` constructor

## Cleanup

Delete stub `doc.go` files from: `pkg/query/`, `pkg/repository/`, `pkg/pagination/`.

## Validation

- `go build ./...` passes
- `go vet ./...` passes
- `go mod tidy` produces clean go.mod/go.sum
- All existing tests pass: `go test ./tests/...`
- New tests added for query builder, pagination, repository helpers, and database system (Phase 5)
