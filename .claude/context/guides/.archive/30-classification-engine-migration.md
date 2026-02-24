# 30 - Classification Engine Database Migration

## Problem Context

Herald's Phase 2 classification engine requires two new tables: `classifications` (1:1 with documents, storing inference results) and `prompts` (named overrides per workflow stage). These tables are the persistence foundation for all subsequent Phase 2 objectives.

## Architecture Approach

Single migration `000002_classification_engine` following the pattern established by `000001_initial_schema`. Both tables are created in the up migration; the down migration drops them in reverse dependency order.

## Implementation

### Step 1: Create the up migration

**New file:** `cmd/migrate/migrations/000002_classification_engine.up.sql`

```sql
CREATE TABLE classifications (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  document_id UUID NOT NULL UNIQUE
    REFERENCES documents(id) ON DELETE CASCADE,
  classification TEXT NOT NULL,
  confidence TEXT NOT NULL
    CHECK (confidence IN ('HIGH', 'MEDIUM', 'LOW')),
  markings_found JSONB NOT NULL DEFAULT '[]',
  rationale TEXT NOT NULL,
  classified_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  model_name TEXT NOT NULL,
  provider_name TEXT NOT NULL,
  validated_by TEXT,
  validated_at TIMESTAMPTZ
);

CREATE INDEX idx_classifications_classification ON classifications(classification);
CREATE INDEX idx_classifications_confidence ON classifications(confidence);
CREATE INDEX idx_classifications_classified_at ON classifications(classified_at DESC);

CREATE TABLE prompts (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT NOT NULL UNIQUE,
  stage TEXT NOT NULL
    CHECK (stage IN ('init', 'classify', 'enhance')),
  system_prompt TEXT NOT NULL,
  description TEXT
);

CREATE INDEX idx_prompts_stage ON prompts(stage);
```

### Step 2: Create the down migration

**New file:** `cmd/migrate/migrations/000002_classification_engine.down.sql`

```sql
DROP TABLE IF EXISTS classifications;
DROP TABLE IF EXISTS prompts;
```

## Validation Criteria

- [ ] `go run ./cmd/migrate/ up` applies cleanly with no errors
- [ ] `go run ./cmd/migrate/ down 1` reverts cleanly (drops both tables)
- [ ] Re-applying after revert succeeds (idempotent up/down cycle)
- [ ] `document_id` UNIQUE constraint enforces 1:1 relationship
- [ ] `ON DELETE CASCADE` removes classification when parent document is deleted
- [ ] `confidence` CHECK rejects values outside HIGH/MEDIUM/LOW
- [ ] `stage` CHECK rejects values outside init/classify/enhance
- [ ] All four indexes are created
