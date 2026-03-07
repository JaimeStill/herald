import { LitElement, html, nothing } from "lit";
import { customElement, property } from "lit/decorators.js";

import badgeStyles from "@styles/badge.module.css";
import styles from "./markings-list.module.css";

/**
 * Pure element that renders an array of security marking strings as
 * badge tags. Displays an empty-state message when no markings are present.
 */
@customElement("hd-markings-list")
export class MarkingsList extends LitElement {
  static styles = [badgeStyles, styles];

  @property({ type: Array }) markings: string[] = [];

  render() {
    if (!this.markings.length) {
      return html`<span class="empty">No markings found</span>`;
    }

    return html`
      <div class="markings">
        ${this.markings.map((m) => html`<span class="badge">${m}</span>`)}
      </div>
    `;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    "hd-markings-list": MarkingsList;
  }
}
