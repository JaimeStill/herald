# 30 - Classification Engine Database Migration

## Summary

Created migration `000002_classification_engine` adding `classifications` and `prompts` tables to the Herald database schema. These tables are the persistence foundation for Phase 2's classification engine — `classifications` stores 1:1 inference results linked to documents, and `prompts` stores named prompt overrides per workflow stage.

## Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Table ordering in up migration | classifications before prompts | classifications has the FK dependency on documents; prompts is independent |
| Down migration ordering | classifications then prompts | Drop FK-dependent table first, though both are independent of each other |
| `classification` column | Unconstrained TEXT | Security marking values may include unexpected levels (per objective #24 decision 9) |
| `confidence` column | CHECK-constrained to HIGH/MEDIUM/LOW | Known enum, categorical scoring aligned with classify-docs |

## Files Modified

- `cmd/migrate/migrations/000002_classification_engine.up.sql` (new)
- `cmd/migrate/migrations/000002_classification_engine.down.sql` (new)

## Patterns Established

- None new — follows the migration pattern from `000001_initial_schema`

## Validation Results

- Migration up applies cleanly (version 2, not dirty)
- Single-step revert (`-steps -1`) reverts to version 1
- Re-apply after revert succeeds
- Constraint validation deferred to API integration testing
