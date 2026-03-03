import { LitElement, html, nothing } from 'lit';
import { customElement, property } from 'lit/decorators.js';
import type { WorkflowStage } from '@app/classifications';
import type { Document } from '@app/documents';
import { formatBytes, formatDate } from '@app/formatting';
import styles from './document-card.module.css';

/**
 * Pure element that displays a document's metadata, classification summary,
 * and action buttons. Dispatches `classify` and `review` custom events.
 */
@customElement('hd-document-card')
export class DocumentCard extends LitElement {
  static styles = styles;

  @property({ type: Object }) document!: Document;
  @property({ type: Boolean }) classifying = false;
  @property() currentNode: WorkflowStage | null = null;
  @property({ type: Array }) completedNodes: WorkflowStage[] = [];

  private get classifyDisabled(): boolean {
    return this.document.status === 'complete' || this.classifying;
  }

  private handleClassify() {
    this.dispatchEvent(new CustomEvent('classify', {
      detail: { id: this.document.id },
      bubbles: true,
      composed: true,
    }));
  }

  private handleReview() {
    this.dispatchEvent(new CustomEvent('review', {
      detail: { id: this.document.id },
      bubbles: true,
      composed: true,
    }));
  }

  private renderClassification() {
    if (!this.document.classification) return nothing;

    return html`
      <div class="classification">
        <span class="classification-label">${this.document.classification}</span>
        ${this.document.confidence
        ? html`<span class="confidence">${this.document.confidence}</span>`
        : nothing}
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
      <div class="card">
        <div class="header">
          <span class="filename">${doc.filename}</span>
          <span class="badge ${doc.status}">${doc.status}</span>
        </div>

        <div class="meta">
          ${doc.page_count !== null
        ? html`<span>${doc.page_count} pages</span>`
        : nothing}
          <span>${formatBytes(doc.size_bytes)}</span>
          <span>${formatDate(doc.uploaded_at)}</span>
        </div>

        ${this.renderClassification()}
        ${this.renderProgress()}

        <div class="actions">
          <button
            class="btn classify-btn"
            ?disabled=${this.classifyDisabled}
            @click=${this.handleClassify}
          >Classify</button>
          <button
            class="btn review-btn"
            @click=${this.handleReview}
          >Review</button>
        </div>
      </div>
    `;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    'hd-document-card': DocumentCard;
  }
}
