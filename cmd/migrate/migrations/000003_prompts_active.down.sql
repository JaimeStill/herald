DROP INDEX IF EXISTS idx_prompts_stage_active;

ALTER TABLE prompts
  DROP COLUMN IF EXISTS active;
