ALTER TABLE prompts DROP CONSTRAINT prompts_stage_check;

ALTER TABLE prompts
  ADD CONSTRAINT prompts_stage_check
  CHECK (stage IN ('classify', 'enhance', 'finalize'));
