# 5 - Database Toolkit

## Problem Context

Herald needs PostgreSQL connection management, SQL query building, repository helpers, and pagination support. These are the `pkg/` building blocks consumed by domain systems (documents, classifications, prompts) in later objectives. All patterns are adapted directly from agent-lab, using `database/sql` as the standard Go database abstraction with pgx as the registered driver via `pgx/v5/stdlib`.

## Architecture Approach

- **Database abstraction**: `database/sql` + pgx stdlib driver — standard interfaces, portable, ecosystem-compatible
- **Cold/hot start**: `New()` calls `sql.Open()` (validates DSN, no connection); `Start()` registers ping + close lifecycle hooks
- **Row scanning**: `ScanFunc[T]` — domain packages define explicit scan functions, no struct tag magic
- **Repository interfaces**: `Querier`, `Executor`, `Scanner` — abstracts `*sql.DB`, `*sql.Tx`, `*sql.Conn` for both normal and transaction contexts
- **Error mapping**: `sql.ErrNoRows` + `pgconn.PgError` code 23505 — domain-agnostic translation
- **Query builder**: ProjectionMap + Builder — separates reusable column definitions from per-query conditions

## Implementation

### Step 1: Add pgx Dependency

Terminal command:

```bash
cd /home/jaime/code/herald
go get github.com/jackc/pgx/v5
```

### Step 2: Database System

**`pkg/database/database.go`** — NEW

```go
package database

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/JaimeStill/herald/pkg/lifecycle"
)

type System interface {
	Connection() *sql.DB
	Start(lc *lifecycle.Coordinator) error
}

type database struct {
	conn        *sql.DB
	logger      *slog.Logger
	connTimeout time.Duration
}

func New(cfg *Config, logger *slog.Logger) (System, error) {
	db, err := sql.Open("pgx", cfg.Dsn())
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetimeDuration())

	return &database{
		conn:        db,
		logger:      logger.With("system", "database"),
		connTimeout: cfg.ConnTimeoutDuration(),
	}, nil
}

func (d *database) Connection() *sql.DB {
	return d.conn
}

func (d *database) Start(lc *lifecycle.Coordinator) error {
	d.logger.Info("starting database connection")

	lc.OnStartup(func() {
		pingCtx, cancel := context.WithTimeout(lc.Context(), d.connTimeout)
		defer cancel()

		if err := d.conn.PingContext(pingCtx); err != nil {
			d.logger.Error("database ping failed", "error", err)
			return
		}

		d.logger.Info("database connection established")
	})

	lc.OnShutdown(func() {
		<-lc.Context().Done()
		d.logger.Info("closing database connection")

		if err := d.conn.Close(); err != nil {
			d.logger.Error("database close failed", "error", err)
			return
		}

		d.logger.Info("database connection closed")
	})

	return nil
}
```

**`pkg/database/errors.go`** — NEW

```go
package database

import "errors"

var ErrNotReady = errors.New("database not ready")
```

### Step 3: Query Builder

Delete `pkg/query/doc.go`, then create these files:

**`pkg/query/projection.go`** — NEW (replaces doc.go)

```go
package query

import (
	"fmt"
	"strings"
)

type ProjectionMap struct {
	schema     string
	table      string
	alias      string
	columns    map[string]string
	columnList []string
}

func NewProjectionMap(schema, table, alias string) *ProjectionMap {
	return &ProjectionMap{
		schema:     schema,
		table:      table,
		alias:      alias,
		columns:    make(map[string]string),
		columnList: make([]string, 0),
	}
}

func (p *ProjectionMap) Project(column, viewName string) *ProjectionMap {
	qualified := fmt.Sprintf("%s.%s", p.alias, column)
	p.columns[viewName] = qualified
	p.columnList = append(p.columnList, qualified)
	return p
}

func (p *ProjectionMap) Alias() string {
	return p.alias
}

func (p *ProjectionMap) Table() string {
	return fmt.Sprintf("%s.%s %s", p.schema, p.table, p.alias)
}

func (p *ProjectionMap) Column(viewName string) string {
	if col, ok := p.columns[viewName]; ok {
		return col
	}
	return viewName
}

func (p *ProjectionMap) Columns() string {
	return strings.Join(p.columnList, ", ")
}

func (p *ProjectionMap) ColumnList() []string {
	return p.columnList
}
```

**`pkg/query/builder.go`** — NEW

```go
package query

import (
	"fmt"
	"reflect"
	"strings"
)

type condition struct {
	clause string
	args   []any
}

type SortField struct {
	Field      string
	Descending bool
}

type Builder struct {
	projection        *ProjectionMap
	conditions        []condition
	orderByFields     []SortField
	defaultSortFields []SortField
}

func NewBuilder(projection *ProjectionMap, defaultSort ...SortField) *Builder {
	return &Builder{
		projection:        projection,
		conditions:        make([]condition, 0),
		defaultSortFields: defaultSort,
	}
}

func ParseSortFields(s string) []SortField {
	if s == "" {
		return nil
	}

	parts := strings.Split(s, ",")
	fields := make([]SortField, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		if after, ok := strings.CutPrefix(part, "-"); ok {
			fields = append(fields, SortField{
				Field:      after,
				Descending: true,
			})
		} else {
			fields = append(fields, SortField{
				Field:      part,
				Descending: false,
			})
		}
	}

	return fields
}

func (b *Builder) Build() (string, []any) {
	where, args, _ := b.buildWhere(1)
	orderBy := b.buildOrderBy()

	sql := fmt.Sprintf(
		"SELECT %s FROM %s%s%s",
		b.projection.Columns(),
		b.projection.Table(),
		where,
		orderBy,
	)

	return sql, args
}

func (b *Builder) BuildCount() (string, []any) {
	where, args, _ := b.buildWhere(1)
	sql := fmt.Sprintf("SELECT COUNT(*) FROM %s%s", b.projection.Table(), where)
	return sql, args
}

func (b *Builder) BuildPage(page, pageSize int) (string, []any) {
	where, args, _ := b.buildWhere(1)
	orderBy := b.buildOrderBy()
	offset := (page - 1) * pageSize

	sql := fmt.Sprintf(
		"SELECT %s FROM %s%s%s LIMIT %d OFFSET %d",
		b.projection.Columns(),
		b.projection.Table(),
		where,
		orderBy,
		pageSize,
		offset,
	)

	return sql, args
}

func (b *Builder) BuildSingle(idField string, id any) (string, []any) {
	col := b.projection.Column(idField)
	sql := fmt.Sprintf(
		"SELECT %s FROM %s WHERE %s = $1",
		b.projection.Columns(),
		b.projection.Table(),
		col,
	)
	return sql, []any{id}
}

func (b *Builder) BuildSingleOrNull() (string, []any) {
	where, args, _ := b.buildWhere(1)
	sql := fmt.Sprintf(
		"SELECT %s FROM %s%s LIMIT 1",
		b.projection.Columns(),
		b.projection.Table(),
		where,
	)
	return sql, args
}

func (b *Builder) OrderByFields(fields []SortField) *Builder {
	b.orderByFields = fields
	return b
}

func (b *Builder) WhereContains(field string, value *string) *Builder {
	if value == nil || *value == "" {
		return b
	}
	col := b.projection.Column(field)
	b.conditions = append(b.conditions, condition{
		clause: fmt.Sprintf("%s ILIKE $%%d", col),
		args:   []any{"%" + *value + "%"},
	})
	return b
}

func (b *Builder) WhereEquals(field string, value any) *Builder {
	if isNil(value) {
		return b
	}
	col := b.projection.Column(field)
	b.conditions = append(b.conditions, condition{
		clause: fmt.Sprintf("%s = $%%d", col),
		args:   []any{value},
	})
	return b
}

func (b *Builder) WhereIn(field string, values []any) *Builder {
	if len(values) == 0 {
		return b
	}
	col := b.projection.Column(field)
	placeholders := make([]string, len(values))
	for i := range values {
		placeholders[i] = "$%d"
	}
	b.conditions = append(b.conditions, condition{
		clause: fmt.Sprintf("%s IN (%s)", col, strings.Join(placeholders, ", ")),
		args:   values,
	})
	return b
}

func (b *Builder) WhereNullable(column string, val any) *Builder {
	col := b.projection.Column(column)
	if isNil(val) {
		b.conditions = append(b.conditions, condition{
			clause: col + " IS NULL",
			args:   nil,
		})
	} else {
		b.conditions = append(b.conditions, condition{
			clause: fmt.Sprintf("%s = $%%d", col),
			args:   []any{val},
		})
	}

	return b
}

func (b *Builder) WhereSearch(search *string, fields ...string) *Builder {
	if search == nil || *search == "" || len(fields) == 0 {
		return b
	}

	clauses := make([]string, len(fields))
	args := make([]any, len(fields))
	searchPattern := "%" + *search + "%"

	for i, field := range fields {
		col := b.projection.Column(field)
		clauses[i] = fmt.Sprintf("%s ILIKE $%%d", col)
		args[i] = searchPattern
	}

	b.conditions = append(b.conditions, condition{
		clause: "(" + strings.Join(clauses, " OR ") + ")",
		args:   args,
	})
	return b
}

func (b *Builder) buildOrderBy() string {
	fields := b.orderByFields
	if len(fields) == 0 {
		fields = b.defaultSortFields
	}

	if len(fields) == 0 {
		return ""
	}

	parts := make([]string, len(fields))
	for i, f := range fields {
		col := b.projection.Column(f.Field)
		dir := "ASC"
		if f.Descending {
			dir = "DESC"
		}
		parts[i] = fmt.Sprintf("%s %s", col, dir)
	}

	return " ORDER BY " + strings.Join(parts, ", ")
}

func (b *Builder) buildWhere(startParam int) (string, []any, int) {
	if len(b.conditions) == 0 {
		return "", nil, startParam
	}

	clauses := make([]string, 0, len(b.conditions))
	args := make([]any, 0)
	paramIdx := startParam

	for _, cond := range b.conditions {
		clause := cond.clause
		for _, arg := range cond.args {
			clause = strings.Replace(clause, "$%d", fmt.Sprintf("$%d", paramIdx), 1)
			args = append(args, arg)
			paramIdx++
		}
		clauses = append(clauses, clause)
	}

	return " WHERE " + strings.Join(clauses, " AND "), args, paramIdx
}

func isNil(value any) bool {
	if value == nil {
		return true
	}

	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Ptr, reflect.Map, reflect.Slice, reflect.Chan, reflect.Func, reflect.Interface:
		return v.IsNil()
	}

	return false
}
```

### Step 4: Repository Helpers

Delete `pkg/repository/doc.go`, then create these files:

**`pkg/repository/repository.go`** — NEW (replaces doc.go)

```go
package repository

import (
	"context"
	"database/sql"
)

type Querier interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type Executor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

type Scanner interface {
	Scan(dest ...any) error
}

type ScanFunc[T any] func(Scanner) (T, error)

func WithTx[T any](ctx context.Context, db *sql.DB, fn func(tx *sql.Tx) (T, error)) (T, error) {
	var zero T

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return zero, err
	}
	defer tx.Rollback()

	result, err := fn(tx)
	if err != nil {
		return zero, err
	}

	if err := tx.Commit(); err != nil {
		return zero, err
	}

	return result, nil
}

func QueryOne[T any](ctx context.Context, q Querier, query string, args []any, scan ScanFunc[T]) (T, error) {
	var zero T
	row := q.QueryRowContext(ctx, query, args...)
	result, err := scan(row)
	if err != nil {
		return zero, err
	}
	return result, nil
}

func QueryMany[T any](ctx context.Context, q Querier, query string, args []any, scan ScanFunc[T]) ([]T, error) {
	rows, err := q.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make([]T, 0)
	for rows.Next() {
		item, err := scan(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}

func ExecExpectOne(ctx context.Context, e Executor, query string, args ...any) error {
	result, err := e.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return sql.ErrNoRows
	}

	return nil
}
```

**`pkg/repository/errors.go`** — NEW

```go
package repository

import (
	"database/sql"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

const pgDuplicateKeyCode = "23505"

func MapError(err error, notFoundErr, duplicateErr error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, sql.ErrNoRows) {
		return notFoundErr
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == pgDuplicateKeyCode {
		return duplicateErr
	}

	return err
}
```

### Step 5: Pagination

Delete `pkg/pagination/doc.go`, then create these files:

**`pkg/pagination/config.go`** — NEW (replaces doc.go)

```go
package pagination

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	DefaultPageSize int `toml:"default_page_size"`
	MaxPageSize     int `toml:"max_page_size"`
}

type ConfigEnv struct {
	DefaultPageSize string
	MaxPageSize     string
}

func (c *Config) Finalize(env *ConfigEnv) error {
	c.loadDefaults()
	if env != nil {
		c.loadEnv(env)
	}
	return c.validate()
}

func (c *Config) Merge(overlay *Config) {
	if overlay.DefaultPageSize != 0 {
		c.DefaultPageSize = overlay.DefaultPageSize
	}
	if overlay.MaxPageSize != 0 {
		c.MaxPageSize = overlay.MaxPageSize
	}
}

func (c *Config) loadDefaults() {
	if c.DefaultPageSize <= 0 {
		c.DefaultPageSize = 20
	}
	if c.MaxPageSize <= 0 {
		c.MaxPageSize = 100
	}
}

func (c *Config) loadEnv(env *ConfigEnv) {
	if env.DefaultPageSize != "" {
		if v := os.Getenv(env.DefaultPageSize); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				c.DefaultPageSize = n
			}
		}
	}
	if env.MaxPageSize != "" {
		if v := os.Getenv(env.MaxPageSize); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				c.MaxPageSize = n
			}
		}
	}
}

func (c *Config) validate() error {
	if c.DefaultPageSize < 1 {
		return fmt.Errorf("default_page_size must be positive")
	}
	if c.MaxPageSize < 1 {
		return fmt.Errorf("max_page_size must be positive")
	}
	if c.DefaultPageSize > c.MaxPageSize {
		return fmt.Errorf("default_page_size cannot exceed max_page_size")
	}
	return nil
}
```

**`pkg/pagination/pagination.go`** — NEW

```go
package pagination

import (
	"encoding/json"
	"net/url"
	"strconv"

	"github.com/JaimeStill/herald/pkg/query"
)

type SortFields []query.SortField

func (s *SortFields) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		*s = query.ParseSortFields(str)
		return nil
	}

	var fields []query.SortField
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	*s = fields
	return nil
}

type PageRequest struct {
	Page     int        `json:"page"`
	PageSize int        `json:"page_size"`
	Search   *string    `json:"search,omitempty"`
	Sort     SortFields `json:"sort,omitempty"`
}

func (r *PageRequest) Normalize(cfg Config) {
	if r.Page < 1 {
		r.Page = 1
	}
	if r.PageSize < 1 {
		r.PageSize = cfg.DefaultPageSize
	}
	if r.PageSize > cfg.MaxPageSize {
		r.PageSize = cfg.MaxPageSize
	}
}

func (r *PageRequest) Offset() int {
	return (r.Page - 1) * r.PageSize
}

func PageRequestFromQuery(values url.Values, cfg Config) PageRequest {
	page, _ := strconv.Atoi(values.Get("page"))
	pageSize, _ := strconv.Atoi(values.Get("page_size"))

	var search *string
	if s := values.Get("search"); s != "" {
		search = &s
	}

	sort := query.ParseSortFields(values.Get("sort"))

	req := PageRequest{
		Page:     page,
		PageSize: pageSize,
		Search:   search,
		Sort:     sort,
	}

	req.Normalize(cfg)
	return req
}

type PageResult[T any] struct {
	Data       []T `json:"data"`
	Total      int `json:"total"`
	Page       int `json:"page"`
	PageSize   int `json:"page_size"`
	TotalPages int `json:"total_pages"`
}

func NewPageResult[T any](data []T, total, page, pageSize int) PageResult[T] {
	totalPages := total / pageSize
	if total%pageSize != 0 {
		totalPages++
	}
	if totalPages < 1 {
		totalPages = 1
	}

	if data == nil {
		data = []T{}
	}

	return PageResult[T]{
		Data:       data,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}
}
```

### Step 6: Cleanup and Verify

Delete the stub doc.go files:

```bash
rm pkg/query/doc.go pkg/repository/doc.go pkg/pagination/doc.go
```

Then verify:

```bash
go mod tidy
go build ./...
go vet ./...
go test ./tests/...
```

## OpenAPI Schema Adjustments

None required. This issue adds only `pkg/`-level infrastructure packages with no HTTP endpoints. The `PageRequest` schema already exists in `pkg/openapi/components.go`. Per-domain `PageResult` schemas will be added when domain handlers define their endpoints in later objectives.

## Validation Criteria

- [ ] `go build ./...` passes
- [ ] `go vet ./...` passes
- [ ] `go mod tidy` produces clean go.mod/go.sum
- [ ] All existing tests pass: `go test ./tests/...`
- [ ] `pkg/query/doc.go` no longer exists (replaced by `projection.go` and `builder.go`)
- [ ] `pkg/repository/doc.go` no longer exists (replaced by `repository.go` and `errors.go`)
- [ ] `pkg/pagination/doc.go` no longer exists (replaced by `config.go` and `pagination.go`)
