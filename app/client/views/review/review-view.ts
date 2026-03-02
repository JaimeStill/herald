import { LitElement, html } from 'lit';
import { customElement, property } from 'lit/decorators.js';
import styles from './review-view.module.css';

@customElement('hd-review-view')
export class ReviewView extends LitElement {
  static styles = styles;

  @property({ type: String }) documentId?: string;

  render() {
    return html`
      <div class="container">
        <h1>Review</h1>
        <p>Classification review for document ${this.documentId ?? 'unknown'}.</p>
      </div>
    `;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    'hd-review-view': ReviewView;
  }
}
