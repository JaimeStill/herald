import { LitElement, html } from 'lit';
import { customElement } from 'lit/decorators.js';
import styles from './prompts-view.module.css';

@customElement('hd-prompts-view')
export class PromptsView extends LitElement {
  static styles = styles;

  render() {
    return html`
      <div class="container">
        <h1>Prompts</h1>
        <p>Prompt management interface.</p>
      </div>
    `;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    'hd-prompts-view': PromptsView;
  }
}
