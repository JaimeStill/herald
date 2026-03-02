import { LitElement, html } from 'lit';
import { customElement } from 'lit/decorators.js';
import styles from './documents-view.module.css';

@customElement('hd-documents-view')
export class DocumentsView extends LitElement {
  static styles = styles;

  render() {
    return html`
      <div class="container">
        <h1>Documents</h1>
        <p>Document management interface.</p>
      </div>
    `;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    'hd-documents-view': DocumentsView;
  }
}
