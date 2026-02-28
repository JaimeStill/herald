# Phase 2 Project Review — Herald

## Context

Phase 2 (Classification Engine) is functionally complete with one committed task remaining (#51 — parallelize classify/enhance nodes). This review identified two additional concerns that should be addressed before moving to Phase 3 (Web Client):

1. **Workflow progress tracking** — no way to observe classification progress through workflow stages
2. **Document filtering by classification metadata** — the web client needs to filter/display documents with their classification data, but a circular dependency prevents documents from importing classifications

The review produced clear resolutions: defer SSE to Phase 3 (capture context in `_project/README.md`), and implement SQL LEFT JOIN support in Phase 2 so documents carry classification metadata without dependency changes.

**Session output**: GitHub issues for new tasks under Objective #27, plus `_project/README.md` updates.

---

## Concern 1: Workflow Progress Tracking — Defer to Phase 3

**Decision**: SSE is the correct protocol. Defer implementation to Phase 3. Capture architectural context in `_project/README.md`.

**Rationale**:
- SSE is unidirectional (server → client), works behind standard load balancers, auto-reconnects natively — correct for classification progress which is purely server-push
- WebSockets add upgrade negotiation, sticky sessions, and message broker requirements for cluster deployments — unjustified complexity for this use case
- go-agents-orchestration already supports observer injection — `workflow.go:45` uses `cfg.Observer = "noop"`, swapping to `NewGraphWithDeps(cfg, observer, checkpointStore)` is a single-line change
- SSE is independently testable via curl without a web client, but the feature's scope (observer infrastructure, event types, async execution, streaming endpoint) belongs in Phase 3 alongside the web client that drives its design
- Phase 2 constraint explicitly listed "No observer/checkpoint infrastructure"

**Phase 3 implementation path**:
1. Port `StreamingObserver` from agent-lab (`~/code/agent-lab/internal/workflows/streaming.go`) — ~60 lines, adapts `observability.Observer` to `chan ExecutionEvent`
2. Define event types: `stage.start`, `stage.complete`, `decision`, `error`, `complete`
3. Change `buildGraph` to accept optional `observability.Observer` parameter
4. Add SSE classify endpoint to classifications handler (sets `text/event-stream` headers, ranges over event channel)
5. Wire observer through from handler → `workflow.Execute`

**Action for Phase 2**: Update `_project/README.md` to capture this decision:
- Add to Key Decisions table: SSE for workflow progress (Phase 3)
- Update "Observability" row to reference SSE streaming observer planned for Phase 3
- Add to Resolved Questions: SSE over WebSockets rationale

### File
- `_project/README.md` — update Key Decisions and Resolved Questions sections

---

## Concern 2: Document Classification JOIN — Phase 2 Implementation

**Decision**: Extend the query builder with JOIN support. Extend `Document` with nullable `Classification`, `Confidence`, and `ClassifiedAt` fields populated via LEFT JOIN. No dependency graph changes.

**Rationale**:
- The web client's primary document list view needs classification metadata for filtering, sorting, and display
- SQL LEFT JOIN on a unique FK (`classifications.document_id`) has negligible overhead
- Documents package defines its own nullable fields (plain `*string`, `*time.Time`) — no import of classifications package needed
- Query builder JOIN support follows the same pattern as the .NET S2VA `ProjectionMap.Join()` with context switching (verified in `~/code/_s2va`)

### Step 1: Extend `pkg/query/ProjectionMap` with JOIN support

**File**: `pkg/query/projection.go`

Add context-switching JOIN capability matching the .NET S2VA pattern:

```go
type joinClause struct {
    joinType string // "LEFT JOIN", "INNER JOIN"
    schema   string
    table    string
    alias    string
    on       string
}
```

- Add `currentAlias string` field — initialized to base table alias, switched after each `.Join()` call
- Store joins in `map[string]joinClause` keyed by alias — enables easy lookup by alias
- `Project()` uses `currentAlias` (instead of hardcoded `alias`) so columns after a `.Join()` map to the joined table
- Add `Join(schema, table, alias, joinType, on string)` method — stores join clause in map and switches `currentAlias`. The join type (LEFT, INNER, etc.) is specified on the clause, keeping the method name simple
- Add `From() string` method — returns `Table()` + all JOIN clauses. When no joins exist, `From() == Table()`

### Step 2: Update `pkg/query/Builder` to use `From()`

**File**: `pkg/query/builder.go`

Replace all `b.projection.Table()` calls with `b.projection.From()` in:
- `Build()` (line 78)
- `BuildCount()` (line 91)
- `BuildPage()` (line 102)
- `BuildSingle()` (line 118)
- `BuildSingleOrNull()` (line 131)

Backward-compatible — `From()` returns the same as `Table()` when no joins are defined.

### Step 3: Extend `Document` struct

**File**: `internal/documents/document.go`

Add three nullable fields:

```go
type Document struct {
    // ... existing 11 fields ...
    Classification *string    `json:"classification,omitempty"`
    Confidence     *string    `json:"confidence,omitempty"`
    ClassifiedAt   *time.Time `json:"classified_at,omitempty"`
}
```

These are `omitempty` — they don't appear in JSON when nil (e.g., Create responses, unclassified documents).

### Step 4: Extend projection and scanDocument directly

**File**: `internal/documents/mapping.go`

Extend the existing `projection` with the LEFT JOIN and classification columns:

```go
var projection = query.
    NewProjectionMap("public", "documents", "d").
    Project("id", "ID").
    // ... all 11 document columns ...
    Join("public", "classifications", "c", "LEFT JOIN", "c.document_id = d.id").
    Project("classification", "Classification").
    Project("confidence", "Confidence").
    Project("classified_at", "ClassifiedAt")
```

Extend `scanDocument` to scan all 14 columns (11 base + 3 joined). The 3 joined fields scan into `*string` / `*time.Time` — NULL from the LEFT JOIN maps to nil naturally.

No separate `detailProjection` or `scanDocumentDetail` needed — this is the initial implementation, not a retrofit.

Extend `Filters` with classification fields:

```go
type Filters struct {
    // ... existing 6 fields ...
    Classification *string `json:"classification,omitempty"`
    Confidence     *string `json:"confidence,omitempty"`
}
```

The `Apply` method adds `WhereEquals("Classification", f.Classification)` and `WhereEquals("Confidence", f.Confidence)` — these resolve to `c.classification` and `c.confidence` via the projection's column map.

### Step 5: Update `FiltersFromQuery` for new filter params

**File**: `internal/documents/mapping.go`

Add parsing for `classification` and `confidence` query parameters in `FiltersFromQuery`.

### Step 6: Update API Cartographer docs

**File**: `_project/api/documents/README.md` and `_project/api/documents/documents.http`

Update the documents endpoint documentation to reflect the new classification fields in List/Find responses and the new filter parameters.

### Step 7: Tests

**Files**: `tests/` (new or extended)

- Query builder tests: verify JOIN SQL generation, column resolution for joined columns, `From()` output with and without joins
- Document mapping tests: verify `scanDocument` handles NULL classification columns correctly

---

## Existing Task: #51 — Parallelize Classify and Enhance Nodes

Already scoped in the GitHub issue. No changes — execute as specified:
- Refactor classify node to use bounded `errgroup` concurrency (remove context accumulation, isolated per-page classification)
- Refactor enhance node to parallelize page rendering and vision calls
- New concurrency limit for inference calls (network-bound, not CPU-bound)

---

## Session Outputs

1. **One GitHub issue** under Objective #27: Query builder JOIN support and document classification view (Steps 1-7). API Cartographer docs and tests are AI responsibilities included in the same task.
2. **Update `_project/README.md` directly** — capture SSE decision context and Phase 3 concerns (fleshed out fully during Phase 3 planning).

Issue #51 (parallelize workflow nodes) already exists.

---

## Verification (post-implementation)

1. `go vet ./...` — passes
2. `go test ./tests/...` — all existing + new tests pass
3. `GET /api/documents` returns documents with `classification`, `confidence`, `classified_at` fields (null for unclassified, populated for classified)
4. `GET /api/documents?classification=SECRET` filters documents by classification
5. `GET /api/documents/{id}` returns document with classification fields
6. `POST /api/classifications/classify/{documentId}` on a multi-page PDF completes in significantly less time than before (parallelization)

---

## Key Files

| File | Change |
|------|--------|
| `pkg/query/projection.go` | Add JOIN support with context switching, `map[string]joinClause` |
| `pkg/query/builder.go` | Use `From()` instead of `Table()` |
| `internal/documents/document.go` | Add 3 nullable classification fields |
| `internal/documents/mapping.go` | Extend projection with JOIN, extend scanDocument, extend Filters |
| `internal/workflow/classify.go` | Parallelize per-page classification (#51) |
| `internal/workflow/enhance.go` | Parallelize page rendering + vision (#51) |
| `_project/README.md` | Capture SSE decision and resolved questions |
| `_project/api/documents/` | Update API docs for new fields/filters |
