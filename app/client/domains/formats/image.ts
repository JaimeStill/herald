import { html } from "lit";

import type { DocumentFormat } from "./format";

/**
 * Raw-image format registration. Covers PNG, JPEG, and WEBP — each renders
 * inline via `<img>`. Because `<img>` carries its intrinsic pixel
 * dimensions, every flex ancestor up to the viewport-sized container must
 * set `min-width: 0` and `min-height: 0` so `object-fit: contain` can scale
 * the image down to the panel's dimensions. See `blob-viewer.module.css`
 * and `review-view.module.css` for the existing cascade.
 */
export const imageFormat: DocumentFormat = {
  id: "image",
  displayName: "Image",
  contentTypes: ["image/png", "image/jpeg", "image/webp"],
  extensions: [".png", ".jpg", ".jpeg", ".webp"],
  renderViewer: (src, title) => html`<img src=${src} alt=${title} />`,
};
