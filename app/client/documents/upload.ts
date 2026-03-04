/** Lifecycle state of a file in the upload queue. */
export type UploadStatus = 'pending' | 'uploading' | 'success' | 'error';

/** A file queued for upload with its metadata and current status. */
export interface UploadEntry {
  file: File;
  status: UploadStatus;
  externalId: number;
  platform: string;
  error?: string;
}
