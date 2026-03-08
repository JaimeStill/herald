import { LitElement, html, nothing } from "lit";
import { customElement, property } from "lit/decorators.js";

import { formatBytes, formatDate } from "@core/formatting";
import type { WorkflowStage } from "@domains/classifications";
import type { Document } from "@domains/documents";

import badgeStyles from "@styles/badge.module.css";
import buttonStyles from "@styles/buttons.module.css";
import cardStyles from "@styles/cards.module.css";
import styles from "./document-card.module.css";

/**
 * Pure element that displays a document's metadata, classification summary,
 * and action buttons. Dispatches `classify` and `review` custom events.
 */
@customElement("hd-document-card")
export class DocumentCard extends LitElement {
  static styles = [buttonStyles, badgeStyles, cardStyles, styles];

  @property({ type: Object }) document!: Document;
  @property({ type: Boolean }) classifying = false;
  @property() currentNode: WorkflowStage | null = null;
  @property({ type: Array }) completedNodes: WorkflowStage[] = [];
  @property({ type: Boolean }) selected = false;

  private get classifyDisabled(): boolean {
    return this.document.status === "complete" || this.classifying;
  }

  private handleClassify() {
    this.dispatchEvent(
      new CustomEvent("classify", {
        detail: { id: this.document.id },
        bubbles: true,
        composed: true,
      }),
    );
  }

  private handleReview() {
    this.dispatchEvent(
      new CustomEvent("review", {
        detail: { id: this.document.id },
        bubbles: true,
        composed: true,
      }),
    );
  }

  private handleDelete() {
    this.dispatchEvent(
      new CustomEvent("delete", {
        detail: { document: this.document },
        bubbles: true,
        composed: true,
      }),
    );
  }

  private handleSelect() {
    this.dispatchEvent(
      new CustomEvent("select", {
        detail: { id: this.document.id },
        bubbles: true,
        composed: true,
      }),
    );
  }

  private renderClassification() {
    if (!this.document.classification) return nothing;

    return html`
      <div class="classification">
        ${this.document.classification} (${this.document.confidence})
      </div>
    `;
  }

  private renderProgress() {
    if (!this.classifying) return nothing;

    return html`
      <hd-classify-progress
        .currentNode=${this.currentNode}
        .completedNodes=${this.completedNodes}
      ></hd-classify-progress>
    `;
  }

  render() {
    const doc = this.document;

    return html`
      <div class="card ${this.selected ? "selected" : ""}">
        <div class="header" @click=${this.handleSelect}>
          <span class="filename">${doc.filename}</span>
          <span class="badge ${doc.status}">${doc.status}</span>
        </div>

        ${this.renderClassification()} ${this.renderProgress()}

        <div class="meta">
          ${doc.page_count !== null
            ? html`<span>${doc.page_count} pages</span>`
            : nothing}
          <span>${formatBytes(doc.size_bytes)}</span>
          <span>${formatDate(doc.uploaded_at)}</span>
          <span>${doc.external_platform} #${doc.external_id}</span>
        </div>

        <div class="actions">
          <button
            class="btn btn-blue"
            ?disabled=${this.classifyDisabled}
            @click=${this.handleClassify}
          >
            Classify
          </button>
          <button class="btn btn-green" @click=${this.handleReview}>
            Review
          </button>
          <button
            class="btn btn-red"
            ?disabled=${this.classifying}
            @click=${this.handleDelete}
          >
            Delete
          </button>
        </div>
      </div>
    `;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    "hd-document-card": DocumentCard;
  }
}
