import type { TemplateResult } from "lit";

/**
 * One supported document format. Registered instances drive the upload
 * widget's accept list, MIME filter, rejection toast, and the blob viewer's
 * render choice. Add a new format by creating a new implementation and
 * registering it in {@link ./registry}.
 */
export interface DocumentFormat {
  /** Short stable identifier for the format (e.g. "pdf", "image"). */
  id: string;
  /** Human-readable label used in drop-zone hints and rejection toasts. */
  displayName: string;
  /** MIME types this format accepts; matched exactly by `findFormat`. */
  contentTypes: string[];
  /** File extensions (with leading dot); joined into the input `accept` attribute. */
  extensions: string[];
  /**
   * Renders the document inline given an object URL or remote src and a
   * title for accessibility. Callers (currently {@link BlobViewer}) treat
   * the return as a standalone template — it must be self-contained.
   */
  renderViewer: (src: string, title: string) => TemplateResult;
}
