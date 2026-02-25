# 37 - Prompts Domain Extensions

## Summary

Extended the prompts domain with hardcoded default instructions and specifications per workflow stage, two new System interface methods (`Instructions`, `Spec`), and two new API endpoints. Removed `StageInit` from the prompts domain entirely since the init workflow node performs image rendering with no LLM interaction. Added a database migration to tighten the stage CHECK constraint to only allow classify and enhance.

## Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Init stage removal | Remove `StageInit` from prompts entirely | Init node is pure image preparation — no LLM calls, no prompt content needed |
| Hardcoded content organization | Separate files (`instructions.go`, `specs.go`) with named constants | Prompt bodies can grow large; constants keep maps clean and files focused |
| Naming convention | `Spec` (abbreviated) consistently across all layers | Aligns file name (`specs.go`), accessor (`Spec()`), interface method, handler method, and URL pattern (`/{stage}/spec`) |
| Instructions fallback | DB active override → hardcoded default | `repo.Instructions` queries for active prompt first, falls back to `Instructions()` package function on `sql.ErrNoRows` |
| Spec sourcing | Pure hardcoded, no DB interaction | Specs are immutable output format contracts; no reason to query DB |

## Files Modified

- `internal/prompts/stages.go` — removed `StageInit`, added `ParseStage`
- `internal/prompts/errors.go` — updated `ErrInvalidStage` message
- `internal/prompts/instructions.go` — new: classify/enhance instruction constants + `Instructions()` accessor
- `internal/prompts/specs.go` — new: classify/enhance spec constants + `Spec()` accessor
- `internal/prompts/system.go` — added `Instructions` and `Spec` to interface
- `internal/prompts/repository.go` — implemented both methods on `repo`
- `internal/prompts/handler.go` — added `StageContent` type, `Instructions`/`Spec` handlers, two new routes
- `cmd/migrate/migrations/000004_prompts_remove_init_stage.up.sql` — new migration
- `cmd/migrate/migrations/000004_prompts_remove_init_stage.down.sql` — new migration
- `_project/api/prompts/README.md` — documented new endpoints, updated stage references
- `_project/api/prompts/prompts.http` — added curl examples for new endpoints
- `tests/prompts/prompts_test.go` — updated for StageInit removal, added ParseStage/Instructions/Spec tests
- `tests/prompts/handler_test.go` — updated mock, added Instructions/Spec handler tests, updated route assertions

## Patterns Established

- **Hardcoded prompt content pattern**: Named constants in dedicated files, unexported map for lookup, exported accessor function with stage validation
- **Stage-scoped content endpoints**: `StageContent` response type with `{stage}` path parameter validated via `ParseStage`
- **DB-with-fallback pattern**: Query for active override, fall back to package-level accessor on `sql.ErrNoRows`

## Validation Results

- `go vet ./...` passes
- All 16 test packages pass
- `go mod tidy` produces no changes
