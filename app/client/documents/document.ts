/** Classification processing state of a document. */
export type DocumentStatus = 'pending' | 'review' | 'complete';

/**
 * Uploaded document with optional classification summary.
 * Mirrors Go `documents.Document` struct. Classification fields
 * are populated from a LEFT JOIN and omitted when unclassified.
 */
export interface Document {
  id: string;
  external_id: number;
  external_platform: string;
  filename: string;
  content_type: string;
  size_bytes: number;
  /** Null when page count has not been determined (e.g., non-PDF). */
  page_count: number | null;
  storage_key: string;
  status: DocumentStatus;
  uploaded_at: string;
  updated_at: string;
  classification?: string;
  confidence?: string;
  classified_at?: string;
}
