import { request, toQueryString } from '@app/core';
import type { PageRequest, PageResult, Result } from '@app/core';
import type { Document } from './document';

const base = '/documents';

/**
 * Stateless API wrapper mirroring the Go documents handler.
 * All methods return {@link Result} — no signals, no state.
 */
export const DocumentService = {
  /** `GET /api/documents` — paginated document list. */
  async list(params?: PageRequest): Promise<Result<PageResult<Document>>> {
    return await request<PageResult<Document>>(
      `${base}${params ? toQueryString(params) : ''}`
    );
  },

  /** `GET /api/documents/:id` — single document by ID. */
  async find(id: string): Promise<Result<Document>> {
    return await request<Document>(`${base}/${id}`);
  },

  /** `POST /api/documents/search` — server-side filtered search. */
  async search(body: PageRequest): Promise<Result<PageResult<Document>>> {
    return await request<PageResult<Document>>(`${base}/search`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    });
  },

  /** `POST /api/documents` — upload a document via multipart form. */
  async upload(
    file: File,
    externalId: number,
    platform: string,
  ): Promise<Result<Document>> {
    const form = new FormData();
    form.append('file', file);
    form.append('external_id', String(externalId));
    form.append('external_platform', platform);

    return await request<Document>(base, {
      method: 'POST',
      body: form,
    });
  },

  /** `DELETE /api/documents/:id` — remove a document and its storage blob. */
  async delete(id: string): Promise<Result<void>> {
    return await request<void>(`${base}/${id}`, {
      method: 'DELETE',
    });
  },
};
