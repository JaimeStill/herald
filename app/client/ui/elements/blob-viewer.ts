import { LitElement, html, nothing } from "lit";
import { customElement, property } from "lit/decorators.js";

import styles from "./blob-viewer.module.css";

/**
 * Generic inline blob viewer. Renders an iframe pointed at the given `src` URL.
 * Caller constructs the URL — the element has no knowledge of API routes or blob types.
 */
@customElement("hd-blob-viewer")
export class BlobViewer extends LitElement {
  static styles = styles;

  @property() override title = "Blob viewer";
  @property() src?: string;

  render() {
    if (!this.src) return nothing;

    return html`<iframe src=${this.src} title=${this.title}></iframe>`;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    "hd-blob-viewer": BlobViewer;
  }
}
