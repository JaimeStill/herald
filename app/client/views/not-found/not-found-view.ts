import { LitElement, html } from 'lit';
import { customElement, property } from 'lit/decorators.js';
import styles from './not-found-view.module.css';

@customElement('hd-not-found-view')
export class NotFoundView extends LitElement {
  static styles = styles;

  @property({ type: String }) path?: string;

  render() {
    return html`
      <div class="container">
        <h1>404</h1>
        <p>Page not found${this.path ? html`: /${this.path}` : ''}</p>
        <a href="">Return home</a>
      </div>
    `;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    'hd-not-found-view': NotFoundView;
  }
}
