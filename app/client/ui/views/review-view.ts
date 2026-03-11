import { LitElement, html, nothing } from "lit";
import { customElement, property, state } from "lit/decorators.js";

import { Auth } from "@core";
import { navigate } from "@core/router";
import type { Document } from "@domains/documents";
import { DocumentService } from "@domains/documents";
import { StorageService } from "@domains/storage";

import buttonStyles from "@styles/buttons.module.css";
import styles from "./review-view.module.css";

/** Route-level view for reviewing a document's classification result. */
@customElement("hd-review-view")
export class ReviewView extends LitElement {
  static styles = [buttonStyles, styles];

  @property() documentId?: string;
  @state() private document?: Document;
  @state() private blobUrl?: string;
  @state() private error?: string;

  async willUpdate(changed: Map<string, unknown>) {
    if (changed.has("documentId") && this.documentId) {
      this.document = undefined;
      this.error = undefined;
      if (this.blobUrl) {
        URL.revokeObjectURL(this.blobUrl);
        this.blobUrl = undefined;
      }

      await this.loadDocument(this.documentId);
    }
  }

  disconnectedCallback() {
    super.disconnectedCallback();
    if (this.blobUrl) {
      URL.revokeObjectURL(this.blobUrl);
    }
  }

  private async loadDocument(documentId: string) {
    const result = await DocumentService.find(documentId);

    if (result.ok) {
      this.document = result.data;
      await this.loadBlob(result.data.storage_key);
    } else {
      this.error = result.error;
    }
  }

  private async loadBlob(storageKey: string) {
    if (!Auth.isEnabled()) {
      this.blobUrl = StorageService.view(storageKey);
      return;
    }

    const result = await StorageService.download(storageKey);
    if (result.ok) {
      this.blobUrl = URL.createObjectURL(result.data);
    }
  }

  private handleBack() {
    navigate("");
  }

  private async handleClassificationChange() {
    if (!this.documentId) return;

    await this.loadDocument(this.documentId);
  }

  render() {
    if (this.error) {
      return html`
        <div class="error">
          <p>${this.error}</p>
          <button class="button" @click=${this.handleBack}>
            Back to Documents
          </button>
        </div>
      `;
    }

    if (!this.document) {
      return html`<div class="loading">Loading document...</div>`;
    }

    return html`
      <div class="panel pdf-panel">
        <hd-blob-viewer
          .title=${this.document.filename}
          .src=${this.blobUrl}
        ></hd-blob-viewer>
      </div>
      <div class="panel classification-panel">
        <hd-classification-panel
          .documentId=${this.documentId ?? ""}
          @validate=${this.handleClassificationChange}
          @update=${this.handleClassificationChange}
        ></hd-classification-panel>
      </div>
    `;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    "hd-review-view": ReviewView;
  }
}
