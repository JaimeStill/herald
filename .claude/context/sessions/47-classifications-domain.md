# 47 - Classifications Domain: Types, System, and Repository

## Summary

Implemented the complete data layer for classification persistence in `internal/classifications/`. Created all domain types (Classification, ValidateCommand, UpdateCommand), sentinel errors with HTTP mapping, the System interface, query mapping with JSONB handling, and the repository with workflow integration. Also moved `workflow/` to `internal/workflow/` to formalize its internal dependency graph.

## Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Confidence type | Plain `string` (not `workflow.Confidence`) | Decouples persistence from workflow types; DB CHECK constraint validates |
| UpdateCommand.UpdatedBy | Included, maps to `validated_by` | Both Validate and Update finalize a classification — consistent tracking |
| Classify status update | Unconditional `SET status = 'review'` | Supports initial classification and re-classification from any state |
| Handler on System | Omitted entirely | Cleaner to add type + interface method together in #48 |
| workflow/ location | Moved to `internal/workflow/` | Already imported internal packages; formalized the dependency |
| collectMarkings | `slices.Sort` + `slices.Compact` | Alphabetized distinct set preferred over discovery-order preservation |
| ErrInvalidStatus HTTP code | 409 Conflict | Request is valid but server state (wrong document status) prevents it |

## Files Modified

- `internal/workflow/` — moved from top-level `workflow/`
- `tests/workflow/types_test.go` — import path update
- `tests/workflow/prompts_test.go` — import path update
- `_project/README.md` — project structure updated for workflow move
- `.claude/CLAUDE.md` — package structure updated for workflow move
- `internal/classifications/doc.go` — deleted (replaced by package comment on classification.go)
- `internal/classifications/classification.go` — new: Classification struct, ValidateCommand, UpdateCommand
- `internal/classifications/errors.go` — new: ErrNotFound, ErrDuplicate, ErrInvalidStatus, MapHTTPStatus
- `internal/classifications/system.go` — new: System interface
- `internal/classifications/mapping.go` — new: projection, Filters, FiltersFromQuery, scanClassification
- `internal/classifications/repository.go` — new: repo struct, New, List, Find, FindByDocument, Classify, Validate, Update, Delete, collectMarkings
- `tests/classifications/classifications_test.go` — new: error mapping, filter parsing, filter application tests

## Patterns Established

- JSONB column handling: scan as `[]byte`, unmarshal to `[]string`, nil guard to `[]string{}`
- Transactional domain+status operations: classification upsert + document status transition in same TX
- Status guard pattern: `WHERE status = 'review'` maps 0 rows to `ErrInvalidStatus`
- Upsert with validation reset: `ON CONFLICT DO UPDATE` nulls `validated_by`/`validated_at`

## Validation Results

- `go vet ./...` — pass
- `go test ./tests/...` — 18/18 suites pass
- `go mod tidy` — no changes
