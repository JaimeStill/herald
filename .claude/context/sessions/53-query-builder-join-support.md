# 53 - Query Builder JOIN Support and Document Classification View

## Summary

Extended the query builder with JOIN support using a context-switching pattern adapted from S2VA's .NET `ProjectionMap.Join()`. Updated the documents domain to include classification metadata via LEFT JOIN, enabling the web client to display and filter documents with their classification status.

## Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| JOIN storage | `map[string]JoinClause` with `Index` field | Map gives O(1) lookup by alias, `Index` preserves insertion order for deterministic SQL via `slices.SortedFunc(maps.Values(...))` |
| Context switching | `currentAlias` field on `ProjectionMap` | After `Join()`, subsequent `Project()` calls map to the joined table's alias automatically — matches S2VA pattern |
| `From()` vs `Table()` | `From()` composes `Table()` + join clauses; backward compatible | When no joins exist, `From()` returns `Table()` — all existing code works unchanged |
| `JoinClause` exported | Exported struct with exported fields | Enables consumers to inspect join metadata via `Joins()` accessor |
| Filter case sensitivity | Kept `WhereEquals` as case-sensitive | Filters will be populated from fetched DB values in the web client, so case will already match |

## Files Modified

- `pkg/query/projection.go` — Added `JoinClause` type, `Join()`, `Joins()`, `From()` methods, `currentAlias`/`joins` fields
- `pkg/query/builder.go` — Replaced `Table()` with `From()` in all 5 build methods
- `internal/documents/document.go` — Added `Classification`, `Confidence`, `ClassifiedAt` nullable fields
- `internal/documents/mapping.go` — Extended projection with LEFT JOIN, extended `scanDocument` to 14 columns, added classification filters
- `tests/query/query_test.go` — Added 10 JOIN tests (From, context switching, Joins ordering, builder methods, joined column filters)

## Patterns Established

- **ProjectionMap JOIN pattern**: `Join(schema, table, alias, joinType, on)` switches `currentAlias` so subsequent `Project()` calls automatically bind to the joined table. Multi-table projections are built with a single fluent chain.
- **Ordered map iteration**: `slices.SortedFunc(maps.Values(m), cmpFunc)` for deterministic iteration of map values by an `Index` field — avoids parallel bookkeeping with a separate order slice.
- **Cross-domain view via LEFT JOIN**: Domains add nullable fields populated by JOINs rather than importing peer packages, avoiding circular dependencies.

## Validation Results

- `go vet ./...` — pass
- `go test ./tests/...` — all 18 packages pass
- `go mod tidy` — no changes
- Manual verification: `GET /api/documents` returns classification fields, filtering by `?confidence=HIGH` works correctly
