# Objective Planning Session — #27 Classifications Domain (Merged)

## Context

Objective #26 (Classification Workflow) is 100% complete — all 5 sub-issues closed. The next objective (#27 Classifications Domain) and #28 (Classification HTTP Endpoints) were originally scoped as separate objectives but should have been one. This session merges #28's scope into #27, deletes #28, transitions out of #26, and decomposes the merged objective into sub-issues.

## Step 0: Transition Closeout

### Status Assessment
- **Objective #26 — Classification Workflow**: 5/5 sub-issues closed (100%)
  - #37 Prompts domain extensions — CLOSED
  - #38 Workflow foundation — CLOSED
  - #39 Init node — CLOSED
  - #40 Classify node — CLOSED
  - #41 Enhance/finalize/graph/Execute — CLOSED
- No incomplete work to carry forward or move to backlog

### Actions
1. Close objective #26
2. Update `_project/phase.md` — mark #26 as Complete, remove #28 row
3. Update issue #27 body — incorporate #28's scope (handler, routes, API wiring, Cartographer docs)
4. Close and delete issue #28
5. Delete `_project/objective.md` (previous objective)

## Step 1: Merged Objective Scope

**Objective #27 — Classifications Domain** now covers the full classifications vertical: persistence layer, business operations (classify/validate/adjust), HTTP endpoints, API wiring, and documentation.

### Architecture Decisions

1. **`Classify` lives on the System interface**: The `classifications.repo` holds `*workflow.Runtime` as a constructor dependency. `Classify(ctx, documentID)` internally calls `workflow.Execute()`, stores the result, and transitions document status. The handler just calls `sys.Classify(ctx, documentID)` — thin and consistent with documents/prompts patterns.

2. **Validate and Update are alternatives, not sequential**: Both transition document status `review → complete`. Validate = human agrees with AI classification. Update = human manually sets `classification` and `rationale` (overwrites the AI values). Re-classification (another `Classify` call) resets validation fields via upsert semantics.

3. **No migration needed**: Update overwrites the existing `classification` and `rationale` columns directly — no separate `adjusted_classification`/`adjustment_rationale` columns.

4. **Workflow runtime assembly in `api.NewDomain`**: Construction order is `documents.System → prompts.System → workflow.Runtime → classifications.System → Domain`. The `workflow.Runtime` is an ephemeral construction dependency, not stored on Domain.

## Step 3: Sub-Issue Decomposition

### Sub-Issue 1: Classifications domain — types, system, and repository

**Scope**: Everything in `internal/classifications/`. Complete data layer with no HTTP surface.

**Files created**:
- `internal/classifications/classification.go` — `Classification`, `ValidateCommand`, `UpdateCommand`
- `internal/classifications/system.go` — `System` interface
- `internal/classifications/errors.go` — sentinel errors + `MapHTTPStatus`
- `internal/classifications/mapping.go` — projection, defaultSort, Filters, FiltersFromQuery, scanClassification
- `internal/classifications/repository.go` — `repo` struct, `New(db, rt, logger, pagination) System`
- Remove `internal/classifications/doc.go` (stub)

**System interface**:
```go
type System interface {
    Handler() *Handler
    List(ctx, page, filters) (*PageResult[Classification], error)
    Find(ctx, id) (*Classification, error)
    FindByDocument(ctx, documentID) (*Classification, error)
    Classify(ctx, documentID) (*Classification, error)
    Validate(ctx, id, cmd ValidateCommand) (*Classification, error)
    Update(ctx, id, cmd UpdateCommand) (*Classification, error)
    Delete(ctx, id) error
}
```

**Key operations**:
- `Classify`: calls `workflow.Execute()` → collects all `MarkingsFound` from pages (deduped) → upserts via `ON CONFLICT (document_id) DO UPDATE` (resets validation fields) → transitions document `pending → review` in same TX
- `Validate`: sets `validated_by`/`validated_at` → transitions document `review → complete`
- `Update`: overwrites `classification` and `rationale` with human-provided values → transitions document `review → complete`

**Tests**: `tests/classifications/classifications_test.go` — error mapping, filter parsing, filter application (same scope as documents/prompts domain tests)

**Labels**: `feature`
**Dependencies**: None (workflow package already complete)

---

### Sub-Issue 2: Classifications handler, API wiring, and API Cartographer docs

**Scope**: HTTP layer, route registration, Domain wiring, API documentation.

**Files created/modified**:
- `internal/classifications/handler.go` — `Handler`, `NewHandler`, `Routes()`, all endpoint methods
- `internal/api/domain.go` — add `Classifications classifications.System`, assemble `workflow.Runtime` in `NewDomain`
- `internal/api/routes.go` — register classifications routes
- `_project/api/classifications/README.md` — API Cartographer docs
- `_project/api/classifications/classifications.http` — `.http` request file

**Routes**:
```
GET    /classifications                     — paginated list with filters
GET    /classifications/{id}                — find by ID
GET    /classifications/document/{id}       — find by document
POST   /classifications/search               — search with JSON body
POST   /classifications/{documentId}        — classify single document
POST   /classifications/{id}/validate       — mark validated
PUT    /classifications/{id}                — update classification manually
DELETE /classifications/{id}                — delete classification
```

**Tests**: `tests/classifications/handler_test.go` — mock System pattern from `tests/documents/handler_test.go` and `tests/prompts/handler_test.go`

**Labels**: `feature`
**Dependencies**: Sub-issue 1

## `_project/objective.md` Content

After sub-issue creation, create `_project/objective.md` documenting the merged objective with scope, sub-issues table, and architecture decisions listed above.

## Execution Order

1. Transition closeout (close #26, update phase.md, merge #28 into #27, delete #28)
2. Create sub-issues, link to #27
3. Add sub-issues to project board, assign Phase 2
4. Create `_project/objective.md`
