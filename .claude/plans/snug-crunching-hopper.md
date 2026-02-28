# Plan: Issue #53 — Query Builder JOIN Support and Document Classification View

## Context

The Phase 3 web client needs to display documents alongside their classification metadata (classification level, confidence, timestamp) and support filtering by those fields. Currently, `Document` and `Classification` are separate types in separate packages with no joined view. A SQL LEFT JOIN approach avoids circular dependencies — the documents domain defines its own nullable fields populated by the join.

This adapts the S2VA `ProjectionMap.Join()` context-switching pattern: after `Join()`, subsequent `Project()` calls map to the joined table's alias automatically.

## Approach

### Step 1: Extend `ProjectionMap` with JOIN support

**File:** `pkg/query/projection.go`

Add context-switching JOIN capability matching the S2VA pattern:

- `joinClause` struct with `joinType`, `schema`, `table`, `alias`, `on` fields
- `currentAlias` field on `ProjectionMap` — initialized to base alias, switched after each `Join()`
- `joins` slice on `ProjectionMap` to accumulate join clauses
- Modify `Project()` to use `currentAlias` instead of `alias`
- `Join(schema, table, alias, joinType, on string) *ProjectionMap` — stores join clause, switches `currentAlias`, returns self for chaining
- `From() string` — returns `Table()` + all join clauses appended. When no joins exist, equals `Table()` (backward compatible)

### Step 2: Update `Builder` to use `From()`

**File:** `pkg/query/builder.go`

Replace all 5 `b.projection.Table()` calls with `b.projection.From()`:
- `Build()` (line 80)
- `BuildCount()` (line 91)
- `BuildPage()` (line 104)
- `BuildSingle()` (line 120)
- `BuildSingleOrNull()` (line 132)

### Step 3: Extend `Document` struct

**File:** `internal/documents/document.go`

Add three nullable fields (all `json:",omitempty"`):
- `Classification *string`
- `Confidence *string`
- `ClassifiedAt *time.Time`

### Step 4: Extend projection, scan, and filters

**File:** `internal/documents/mapping.go`

- Extend `projection` with `.Join("public", "classifications", "c", "LEFT JOIN", "d.id = c.document_id")` followed by `.Project("classification", "Classification")`, `.Project("confidence", "Confidence")`, `.Project("classified_at", "ClassifiedAt")`
- Extend `scanDocument` to scan 14 columns (11 base + 3 nullable from join)
- Add `Classification *string` and `Confidence *string` to `Filters` struct
- Add query parameter parsing for `classification` and `confidence` in `FiltersFromQuery`
- Add `WhereEquals` calls for both new filters in `Filters.Apply()`

## Verification

- `go vet ./...` passes
- `go test ./tests/...` passes (existing + new tests for JOIN SQL generation)
- `GET /api/documents` returns documents with classification fields (null for unclassified)
- `GET /api/documents?classification=SECRET` filters correctly
- `GET /api/documents/{id}` returns document with classification fields
