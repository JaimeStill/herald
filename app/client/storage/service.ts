import { request } from '@app/core';
import type { Result } from '@app/core';
import type { BlobMeta, BlobList } from './blob';

/** Filtering parameters for blob listing. */
export interface StorageListParams {
  prefix?: string;
  marker?: string;
  max_results?: number;
}

const base = '/storage';

/**
 * Stateless API wrapper mirroring the Go storage handler.
 * All methods return {@link Result} — no signals, no state.
 */
export const StorageService = {
  /** `GET /api/storage` — list blobs with optional prefix and pagination. */
  async list(params?: StorageListParams): Promise<Result<BlobList>> {
    const entries = Object.entries(params ?? {})
      .filter(([, v]) => v !== undefined && v !== null && v !== '')
      .map(([k, v]) => `${encodeURIComponent(k)}=${encodeURIComponent(String(v))}`);

    const qs = entries.length > 0 ? `?${entries.join('&')}` : '';
    return await request<BlobList>(`${base}${qs}`);
  },

  /** `GET /api/storage/:key` — blob metadata by storage key. */
  async find(key: string): Promise<Result<BlobMeta>> {
    return await request<BlobMeta>(`${base}/${key}`);
  },

  /** `GET /api/storage/download/:key` — download blob content. */
  async download(key: string): Promise<Result<Blob>> {
    return await request<Blob>(
      `${base}/download/${key}`,
      undefined,
      (res) => res.blob(),
    );
  },
};
