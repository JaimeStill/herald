import { LitElement, html, nothing } from "lit";
import { customElement, property, state } from "lit/decorators.js";

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
  @state() private error?: string;

  async willUpdate(changed: Map<string, unknown>) {
    if (changed.has("documentId") && this.documentId) {
      this.document = undefined;
      this.error = undefined;

      const result = await DocumentService.find(this.documentId);

      if (result.ok) {
        this.document = result.data;
      } else {
        this.error = result.error;
      }
    }
  }

  private handleBack() {
    navigate("");
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
          .src=${StorageService.view(this.document.storage_key)}
        ></hd-blob-viewer>
      </div>
      <div class="panel classification-panel">
        <h2>${this.document.filename}</h2>
        <p class="status">${this.document.status}</p>
      </div>
    `;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    "hd-review-view": ReviewView;
  }
}
