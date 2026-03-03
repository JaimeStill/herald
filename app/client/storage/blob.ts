/** Metadata for a blob in object storage. Mirrors Go `storage.BlobMeta`. */
export interface BlobMeta {
  name: string;
  content_type: string;
  content_length: number;
  etag: string;
  created_at: string;
  last_modified: string;
}

/** Paginated blob listing with optional continuation marker. */
export interface BlobList {
  blobs: BlobMeta[];
  next_marker?: string;
}
