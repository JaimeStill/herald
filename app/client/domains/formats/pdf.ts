import { html } from "lit";

import type { DocumentFormat } from "./format";

/**
 * PDF format registration. Renders via an iframe so the browser's built-in
 * PDF viewer handles paging, zoom, text selection, and search. Iframes have
 * a fixed default intrinsic size, which makes them trivial to size with
 * flex — no `min-width: 0` cascade is required.
 */
export const pdfFormat: DocumentFormat = {
  id: "pdf",
  displayName: "PDF",
  contentTypes: ["application/pdf"],
  extensions: [".pdf"],
  renderViewer: (src, title) =>
    html`<iframe src=${src} title=${title}></iframe>`,
};
