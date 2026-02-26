# Issue #47 — Classifications Domain: Types, System, and Repository

## Context

Part of Objective #27 (Classifications Domain), Phase 2 (Classification Engine). Implements the complete data layer for classification persistence — all types, errors, query mapping, the System interface, and the repository implementation. The repository holds a `*workflow.Runtime` dependency to support the `Classify` method which calls `workflow.Execute()` internally.

## Files

Delete `internal/classifications/doc.go` (existing stub). Create 5 files in `internal/classifications/`:

| File | Purpose |
|------|---------|
| `classification.go` | `Classification` struct, `ValidateCommand`, `UpdateCommand` |
| `errors.go` | `ErrNotFound`, `ErrDuplicate`, `ErrInvalidStatus` + `MapHTTPStatus` |
| `system.go` | `System` interface + placeholder `Handler` type |
| `mapping.go` | projection, defaultSort, `Filters`, `FiltersFromQuery`, `scanClassification` |
| `repository.go` | `repo` struct, `New()` constructor, all method implementations |

## Implementation Steps

### Step 0: Move `workflow/` → `internal/workflow/`

The workflow package already imports `internal/documents` and `internal/prompts`, making it effectively internal. Now that `internal/classifications` will import it, formalize this by moving it under `internal/`.

1. `mv workflow/ internal/workflow/`
2. Update imports in:
   - `tests/workflow/types_test.go` — `github.com/JaimeStill/herald/workflow` → `github.com/JaimeStill/herald/internal/workflow`
   - `tests/workflow/prompts_test.go` — same import update
3. Update `_project/README.md` project structure to reflect the new location

### Step 1: classification.go

- Package doc comment on this file (replaces `doc.go`)
- `Classification` struct: mirrors DB schema — `ID`, `DocumentID`, `Classification`, `Confidence` (string, not workflow.Confidence), `MarkingsFound` ([]string), `Rationale`, `ClassifiedAt`, `ModelName`, `ProviderName`, `ValidatedBy` (*string), `ValidatedAt` (*time.Time)
- `ValidateCommand`: `ValidatedBy string`
- `UpdateCommand`: `Classification string`, `Rationale string`, `UpdatedBy string` — maps to `validated_by` column since both Validate and Update transition to complete and need to record who finalized

### Step 2: errors.go

- `ErrNotFound` — classification not found
- `ErrDuplicate` — classification already exists
- `ErrInvalidStatus` — document is not in review status (used by Validate/Update when document status prevents transition)
- `MapHTTPStatus`: NotFound→404, Duplicate→409, InvalidStatus→409, default→500

### Step 3: system.go

- `Handler struct{}` — placeholder, filled by #48
- `System` interface: `Handler()`, `List()`, `Find()`, `FindByDocument()`, `Classify()`, `Validate()`, `Update()`, `Delete()`

### Step 4: mapping.go

- **Projection**: table `classifications`, alias `c`, all 11 columns
- **DefaultSort**: `ClassifiedAt DESC`
- **Filters**: `Classification` (*string, equals), `Confidence` (*string, equals), `DocumentID` (*uuid.UUID, equals), `ValidatedBy` (*string, equals)
- **FiltersFromQuery**: parse URL query params, uuid.Parse for document_id
- **scanClassification**: scan `markings_found` as `[]byte` → `json.Unmarshal` into `[]string`, nil guard to `[]string{}`

### Step 5: repository.go

**repo struct**: `db *sql.DB`, `rt *workflow.Runtime`, `logger *slog.Logger`, `pagination pagination.Config`

**Constructor**: `New(db, rt, logger, pagination) System`

**Methods**:
- `Handler()` → returns nil (placeholder)
- `List()` → documents.List pattern, search on Classification + Rationale
- `Find(id)` → BuildSingle("ID", id)
- `FindByDocument(documentID)` → BuildSingle("DocumentID", documentID) (unique FK)
- `Classify(documentID)` →
  1. `workflow.Execute(ctx, r.rt, documentID)`
  2. `collectMarkings()` — dedup across all pages preserving order
  3. Marshal markings to JSON
  4. TX: upsert via `ON CONFLICT (document_id) DO UPDATE` (resets validated_by/validated_at to NULL, sets classified_at = NOW()), then `UPDATE documents SET status = 'review'` (unconditional — supports re-classification from any state)
  5. model_name from `r.rt.Agent.Model.Name`, provider_name from `r.rt.Agent.Provider.Name`
- `Validate(id, cmd)` → TX: update classification validated_by/validated_at, then transition document `review → complete` (WHERE status = 'review', map 0 rows to ErrInvalidStatus)
- `Update(id, cmd)` → TX: overwrite classification/rationale + set validated_by/validated_at, then transition document `review → complete` (same status check)
- `Delete(id)` → ExecExpectOne DELETE (no document status transition)
- `collectMarkings()` — unexported helper, dedup markings preserving discovery order

## Key Design Decisions

1. **Confidence as string**: Decouples persistence from workflow types. DB CHECK constraint validates.
2. **UpdateCommand.UpdatedBy**: Both Validate and Update finalize a classification. `validated_by`/`validated_at` always record who and when, regardless of method.
3. **Classify status update unconditional**: `UPDATE documents SET status = 'review' WHERE id = $1` — no current-status check. Supports initial classification (pending→review) and re-classification (review/complete→review).
4. **Validate/Update status guard**: `WHERE status = 'review'` on document update. If document isn't in review, TX rolls back both changes and returns ErrInvalidStatus.
5. **Delete doesn't transition status**: Deleting a classification doesn't change the document. Re-classification later creates a new record.

## Reference Files

- `internal/documents/repository.go` — repo pattern
- `internal/documents/mapping.go` — projection/filters pattern
- `internal/workflow/types.go` — ClassificationState, ClassificationPage (after move)
- `internal/workflow/workflow.go` — Execute function, WorkflowResult (after move)
- `pkg/repository/repository.go` — QueryOne, QueryMany, WithTx, ExecExpectOne

## Validation

- [ ] Workflow move: `go build ./internal/workflow/...` compiles
- [ ] All tests pass after move: `mise run test`
- [ ] All 5 classification files compile: `go build ./internal/classifications/...`
- [ ] `mise run vet` passes
- [ ] Types match DB schema from migration 000002
- [ ] Upsert uses ON CONFLICT with validation field reset
- [ ] Document status transitions are correct (Classify→review, Validate/Update→complete)
