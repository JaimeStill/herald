import { LitElement, html } from "lit";
import { customElement, property } from "lit/decorators.js";

import buttonStyles from "@styles/buttons.module.css";
import styles from "./pagination-controls.module.css";

/** Pure element that renders prev/next pagination controls with a page indicator. */
@customElement("hd-pagination")
export class PaginationControls extends LitElement {
  static styles = [buttonStyles, styles];

  @property({ type: Number }) page = 1;

  @property({ type: Number, attribute: "total-pages" })
  totalPages = 1;

  private handlePrev() {
    if (this.page > 1) {
      this.dispatchEvent(
        new CustomEvent("page-change", {
          detail: { page: this.page - 1 },
          bubbles: true,
          composed: true,
        }),
      );
    }
  }

  private handleNext() {
    if (this.page < this.totalPages) {
      this.dispatchEvent(
        new CustomEvent("page-change", {
          detail: { page: this.page + 1 },
          bubbles: true,
          composed: true,
        }),
      );
    }
  }

  render() {
    return html`
      <div class="pagination">
        <button
          class="btn"
          ?disabled=${this.page <= 1}
          @click=${this.handlePrev}
        >
          Prev
        </button>
        <span class="page-indicator">
          Page ${this.page} of ${this.totalPages}
        </span>
        <button
          class="btn"
          ?disabled=${this.page >= this.totalPages}
          @click=${this.handleNext}
        >
          Next
        </button>
      </div>
    `;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    "hd-pagination": PaginationControls;
  }
}
