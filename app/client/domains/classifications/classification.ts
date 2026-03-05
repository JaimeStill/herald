/** A stage in the classification workflow pipeline. */
export type WorkflowStage = "init" | "classify" | "enhance" | "finalize";

/** Ordered list of all classification workflow stages. */
export const WORKFLOW_STAGES: readonly WorkflowStage[] = [
  "init",
  "classify",
  "enhance",
  "finalize",
];

/**
 * Classification result for a document.
 * Mirrors Go `classifications.Classification` struct.
 * Validation fields are omitted when the classification has not been validated.
 */
export interface Classification {
  id: string;
  document_id: string;
  classification: string;
  confidence: string;
  markings_found: string[];
  rationale: string;
  classified_at: string;
  model_name: string;
  provider_name: string;
  validated_by?: string;
  validated_at?: string;
}
