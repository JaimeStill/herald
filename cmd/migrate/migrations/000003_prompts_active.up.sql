ALTER TABLE prompts
  ADD COLUMN active BOOLEAN NOT NULL DEFAULT false;

CREATE UNIQUE INDEX idx_prompts_stage_active
  ON prompts(stage)
  WHERE active = true;
