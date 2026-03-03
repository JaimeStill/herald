import { request, toQueryString } from '@app/core';

import type {
  PageRequest,
  PageResult,
  Result
} from '@app/core';

import type {
  Prompt,
  PromptStage,
  StageContent,
  CreatePromptCommand,
  UpdatePromptCommand
} from './prompt';

const base = '/prompts';

/**
 * Stateless API wrapper mirroring the Go prompts handler.
 * All methods return {@link Result} — no signals, no state.
 */
export const PromptService = {
  /** `GET /api/prompts` — paginated prompt list. */
  async list(params?: PageRequest): Promise<Result<PageResult<Prompt>>> {
    return await request<PageResult<Prompt>>(
      `${base}${params ? toQueryString(params) : ''}`
    );
  },

  /** `GET /api/prompts/stages` — available workflow stages. */
  async stages(): Promise<Result<PromptStage[]>> {
    return await request<PromptStage[]>(`${base}/stages`);
  },

  /** `GET /api/prompts/:id` — single prompt by ID. */
  async find(id: string): Promise<Result<Prompt>> {
    return await request<Prompt>(`${base}/${id}`);
  },

  /** `GET /api/prompts/:stage/instructions` — assembled instructions for a stage. */
  async instructions(stage: PromptStage): Promise<Result<StageContent>> {
    return await request<StageContent>(`${base}/${stage}/instructions`);
  },

  /** `GET /api/prompts/:stage/spec` — assembled spec for a stage. */
  async spec(stage: PromptStage): Promise<Result<StageContent>> {
    return await request<StageContent>(`${base}/${stage}/spec`);
  },

  /** `POST /api/prompts/search` — server-side filtered search. */
  async search(body: PageRequest): Promise<Result<PageResult<Prompt>>> {
    return await request<PageResult<Prompt>>(`${base}/search`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    });
  },

  /** `POST /api/prompts` — create a new prompt. */
  async create(command: CreatePromptCommand): Promise<Result<Prompt>> {
    return await request<Prompt>(base, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(command),
    });
  },

  /** `PUT /api/prompts/:id` — update an existing prompt. */
  async update(id: string, command: UpdatePromptCommand): Promise<Result<Prompt>> {
    return await request<Prompt>(`${base}/${id}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(command),
    });
  },

  /** `DELETE /api/prompts/:id` — remove a prompt. */
  async delete(id: string): Promise<Result<void>> {
    return await request<void>(`${base}/${id}`, {
      method: 'DELETE',
    });
  },

  /** `POST /api/prompts/:id/activate` — activate a prompt for its stage. */
  async activate(id: string): Promise<Result<Prompt>> {
    return await request<Prompt>(`${base}/${id}/activate`, {
      method: 'POST',
    });
  },

  /** `POST /api/prompts/:id/deactivate` — deactivate a prompt. */
  async deactivate(id: string): Promise<Result<Prompt>> {
    return await request<Prompt>(`${base}/${id}/deactivate`, {
      method: 'POST',
    });
  },
};
