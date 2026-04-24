import type { DocumentFormat } from "./format";
import { imageFormat } from "./image";
import { pdfFormat } from "./pdf";

/**
 * Frozen list of registered formats. Iteration order is stable — it drives
 * the order of extensions in `accept`, the content types in toasts, and
 * the label list in drop-zone hints.
 */
export const formats: readonly DocumentFormat[] = [imageFormat, pdfFormat];

/**
 * Returns the DocumentFormat whose `contentTypes` list includes the given
 * MIME type, or `undefined` when nothing matches. Used by `blob-viewer` to
 * pick a render strategy and by `document-upload` to filter accepted files.
 */
export function findFormat(contentType?: string): DocumentFormat | undefined {
  if (!contentType) return undefined;
  return formats.find((f) => f.contentTypes.includes(contentType));
}

/**
 * Convenience predicate for the upload widget's file filter — returns true
 * iff `findFormat` would return a registered handler.
 */
export function isSupported(contentType?: string): boolean {
  return findFormat(contentType) !== undefined;
}

/**
 * Builds the comma-separated extension list suitable for an `<input
 * type="file" accept="...">` attribute. Extensions from multiple formats
 * are flattened in registration order.
 */
export function acceptAttribute(): string {
  return formats.flatMap((f) => f.extensions).join(",");
}

/**
 * Returns every registered MIME type. Useful for rejection messages that
 * want to cite the full supported set.
 */
export function allSupportedContentTypes(): string[] {
  return formats.flatMap((f) => f.contentTypes);
}

/**
 * Human-readable label for the upload drop zone, assembled from
 * `displayName`s. Example: "Drag Images / PDFs here or click to browse".
 */
export function dropZoneText(): string {
  const label = formats.map((f) => `${f.displayName}s`).join(" / ");
  return `Drag ${label} here or click to browse`;
}
