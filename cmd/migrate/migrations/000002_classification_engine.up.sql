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
  instructions TEXT NOT NULL,
  description TEXT
);

CREATE INDEX idx_prompts_stage ON prompts(stage);
