# Issue #30 — Classification Engine Database Migration

## Context

Second sub-issue of Objective #24. Herald needs `classifications` and `prompts` tables to support the Phase 2 classification engine. The issue specifies exact schemas, constraints, and indexes.

## Approach

Create migration `000002_classification_engine` following the pattern established by `000001_initial_schema`.

### Files

- `cmd/migrate/migrations/000002_classification_engine.up.sql` — new
- `cmd/migrate/migrations/000002_classification_engine.down.sql` — new

### Up Migration

1. Create `classifications` table:
   - `id UUID PK DEFAULT gen_random_uuid()`
   - `document_id UUID NOT NULL UNIQUE FK → documents(id) ON DELETE CASCADE`
   - `classification TEXT NOT NULL`
   - `confidence TEXT NOT NULL CHECK ('HIGH','MEDIUM','LOW')`
   - `markings_found JSONB NOT NULL DEFAULT '[]'`
   - `rationale TEXT NOT NULL`
   - `classified_at TIMESTAMPTZ NOT NULL DEFAULT NOW()`
   - `model_name TEXT NOT NULL`
   - `provider_name TEXT NOT NULL`
   - `validated_by TEXT`
   - `validated_at TIMESTAMPTZ`

2. Create `prompts` table:
   - `id UUID PK DEFAULT gen_random_uuid()`
   - `name TEXT NOT NULL UNIQUE`
   - `stage TEXT NOT NULL CHECK ('init','classify','enhance')`
   - `system_prompt TEXT NOT NULL`
   - `description TEXT`

3. Create indexes:
   - `idx_classifications_classification` on `classification`
   - `idx_classifications_confidence` on `confidence`
   - `idx_classifications_classified_at` on `classified_at DESC`
   - `idx_prompts_stage` on `stage`

### Down Migration

Drop tables in reverse dependency order: `classifications` first (depends on `documents`), then `prompts`.

## Verification

```bash
# Apply migration
mise run dev  # or manually: go run ./cmd/migrate/ up

# Verify tables exist and constraints work
# Revert migration
go run ./cmd/migrate/ down 1
```
