import { LitElement, html, nothing } from "lit";
import { customElement, state } from "lit/decorators.js";

import buttonStyles from "@styles/buttons.module.css";
import styles from "./documents-view.module.css";

/** Route-level view that composes the document upload and grid modules. */
@customElement("hd-documents-view")
export class DocumentsView extends LitElement {
  static styles = [buttonStyles, styles];

  @state() private showUpload = false;

  private handleUploadComplete() {
    this.showUpload = false;
    this.renderRoot.querySelector("hd-document-grid")?.refresh();
  }

  render() {
    return html`
      <div class="view">
        <div class="view-header">
          <h1>Documents</h1>
          <button
            class="btn upload-toggle"
            @click=${() => (this.showUpload = !this.showUpload)}
          >
            ${this.showUpload ? "Close" : "Upload"}
          </button>
        </div>
        ${this.showUpload
          ? html`
              <hd-document-upload
                @upload-complete=${this.handleUploadComplete}
              ></hd-document-upload>
            `
          : nothing}
        <hd-document-grid></hd-document-grid>
      </div>
    `;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    "hd-documents-view": DocumentsView;
  }
}
