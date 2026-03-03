/** Classification workflow stage that a prompt targets. */
export type PromptStage = 'classify' | 'enhance' | 'finalize';

/**
 * Prompt template used by the classification workflow.
 * Mirrors Go `prompts.Prompt` struct.
 */
export interface Prompt {
  id: string;
  name: string;
  stage: PromptStage;
  instructions: string;
  description?: string;
  active: boolean;
}

/** Assembled prompt content for a specific workflow stage. */
export interface StageContent {
  stage: PromptStage;
  content: string;
}

/** Payload for creating a new prompt. */
export interface CreatePromptCommand {
  name: string;
  stage: PromptStage;
  instructions: string;
  description?: string;
}

/** Payload for updating an existing prompt. */
export interface UpdatePromptCommand {
  name: string;
  stage: PromptStage;
  instructions: string;
  description?: string;
}
