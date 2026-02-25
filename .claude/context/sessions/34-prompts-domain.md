# 34 - Prompts Domain Implementation

## Summary

Implemented the full CRUD domain for named prompt instruction overrides following the documents domain pattern. Each prompt targets a workflow stage (init/classify/enhance) with tunable instructions. An `active` boolean with a partial unique index enforces at most one active prompt per stage, and atomic activation swaps the active prompt within a transaction. Also restructured API Cartographer from flat markdown files to subdirectory-per-group layout with Kulala-compatible `.http` test files.

## Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Stage validation | `UnmarshalJSON` on typed `Stage` string | Rejects invalid stages at JSON decode time — earliest point of failure with clear error messages |
| Activation model | Separate Activate/Deactivate endpoints | Explicit intent over toggle; atomic transaction swaps active prompt per stage |
| Partial unique index | `idx_prompts_stage_active ON prompts(stage) WHERE active = true` | PostgreSQL-native enforcement of at most one active prompt per stage |
| Stage type | Typed string enum (`type Stage string`) | Type safety with exported constants; `slices.Contains` for validation in `UnmarshalJSON` |
| File organization | Smaller focused files (`stages.go`, `errors.go`, `mapping.go`, etc.) | Clear responsibility per file over Go's typical larger-file convention |
| API Cartographer restructure | Subdirectory per group with `README.md` + `[group].http` | Isolates REST testing infrastructure from documentation; supports Kulala.nvim workflow |

## Files Modified

- `cmd/migrate/migrations/000003_prompts_active.up.sql` (new)
- `cmd/migrate/migrations/000003_prompts_active.down.sql` (new)
- `internal/prompts/prompt.go` (new, replaces doc.go)
- `internal/prompts/stages.go` (new)
- `internal/prompts/errors.go` (new)
- `internal/prompts/mapping.go` (new)
- `internal/prompts/system.go` (new)
- `internal/prompts/repository.go` (new)
- `internal/prompts/handler.go` (new)
- `internal/api/domain.go` (modified)
- `internal/api/routes.go` (modified)
- `tests/prompts/prompts_test.go` (new)
- `tests/prompts/handler_test.go` (new)
- `_project/api/README.md` (modified — subdirectory links)
- `_project/api/root.http` (new)
- `_project/api/http-client.env.json` (new)
- `_project/api/prompts/README.md` (new)
- `_project/api/prompts/prompts.http` (new)
- `_project/api/documents/README.md` (moved from documents.md)
- `_project/api/documents/documents.http` (new)
- `_project/api/storage/README.md` (moved from storage.md)
- `_project/api/storage/storage.http` (new)
- `.claude/skills/api-cartographer/SKILL.md` (modified — subdirectory layout, .http conventions, root.http)

## Patterns Established

- **Typed string enum with `UnmarshalJSON`**: Validates constrained string values at JSON decode time, eliminating separate validation functions and repository-layer checks
- **API Cartographer subdirectory layout**: Each route group gets a directory with `README.md` (documentation) and `[group].http` (Kulala test file); root-level endpoints get `root.http`
- **Kulala `.http` convention**: `{{HOST}}` from shared `http-client.env.json`, `###` separators with descriptive names, `@variable` for replaceable IDs

## Validation Results

- 40 tests passing across `tests/prompts/` (domain + handler tests)
- Full test suite (16 packages) passing
- `go vet ./...` clean
- `go mod tidy` no changes
- All 9 endpoints manually verified via curl
