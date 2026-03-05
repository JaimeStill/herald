import { request, stream, toQueryString } from "@core";
import type { PageRequest, PageResult, Result, StreamOptions } from "@core";

import type { Classification } from "./classification";

/** Payload for marking a classification as validated. */
export interface ValidateCommand {
  validated_by: string;
}

/** Payload for manually updating a classification result. */
export interface UpdateCommand {
  classification: string;
  rationale: string;
  updated_by: string;
}

const base = "/classifications";

/**
 * Stateless API wrapper mirroring the Go classifications handler.
 * Request-response methods return {@link Result}. The `classify` streaming
 * method returns an {@link AbortController} for cancellation.
 */
export const ClassificationService = {
  /** `GET /api/classifications` — paginated classification list. */
  async list(
    params?: PageRequest,
  ): Promise<Result<PageResult<Classification>>> {
    return await request<PageResult<Classification>>(
      `${base}${params ? toQueryString(params) : ""}`,
    );
  },

  /** `GET /api/classifications/:id` — single classification by ID. */
  async find(id: string): Promise<Result<Classification>> {
    return await request<Classification>(`${base}/${id}`);
  },

  /** `GET /api/classifications/document/:documentId` — classification for a specific document. */
  async findByDocument(documentId: string): Promise<Result<Classification>> {
    return await request<Classification>(`${base}/document/${documentId}`);
  },

  /** `POST /api/classifications/search` — server-side filtered search. */
  async search(body: PageRequest): Promise<Result<PageResult<Classification>>> {
    return await request<PageResult<Classification>>(`${base}/search`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    });
  },

  /**
   * `POST /api/classifications/:documentId` — run the classification workflow.
   * Returns an SSE event stream. The caller provides {@link StreamOptions}
   * callbacks to process `node.start`, `node.complete`, `complete`, and `error` events.
   */
  classify(documentId: string, options: StreamOptions): AbortController {
    return stream(`${base}/${documentId}`, options, { method: "POST" });
  },

  /** `POST /api/classifications/:id/validate` — mark a classification as human-validated. */
  async validate(
    id: string,
    command: ValidateCommand,
  ): Promise<Result<Classification>> {
    return await request<Classification>(`${base}/${id}/validate`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(command),
    });
  },

  /** `PUT /api/classifications/:id` — manually update a classification result. */
  async update(
    id: string,
    command: UpdateCommand,
  ): Promise<Result<Classification>> {
    return await request<Classification>(`${base}/${id}`, {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(command),
    });
  },

  /** `DELETE /api/classifications/:id` — remove a classification record. */
  async delete(id: string): Promise<Result<void>> {
    return await request<void>(`${base}/${id}`, {
      method: "DELETE",
    });
  },
};
