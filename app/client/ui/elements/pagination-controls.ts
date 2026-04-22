import { LitElement, html } from "lit";
import { customElement, property } from "lit/decorators.js";

import buttonStyles from "@styles/buttons.module.css";
import inputStyles from "@styles/inputs.module.css";
import styles from "./pagination-controls.module.css";

/**
 * Pure element that renders the list-view footer: a per-page size selector,
 * prev/next buttons, and an editable page-number input clamped to the valid range.
 * Dispatches `page-change` and `page-size-change` custom events.
 */
@customElement("hd-pagination")
export class PaginationControls extends LitElement {
  static styles = [buttonStyles, inputStyles, styles];

  @property({ type: Number }) page = 1;

  @property({ type: Number, attribute: "total-pages" })
  totalPages = 1;

  @property({ type: Number }) size = 12;

  @property({ type: Array, attribute: "size-options" })
  sizeOptions: number[] = [12, 24, 48, 96];

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

  private handleSizeChange(e: Event) {
    const target = e.target as HTMLSelectElement;
    this.dispatchEvent(
      new CustomEvent("page-size-change", {
        detail: { size: Number(target.value) },
        bubbles: true,
        composed: true,
      }),
    );
  }

  private handlePageInput(e: Event) {
    const target = e.target as HTMLInputElement;
    const raw = target.valueAsNumber;

    if (!Number.isFinite(raw)) {
      target.value = String(this.page);
      return;
    }

    const clamped = Math.min(Math.max(Math.trunc(raw), 1), this.totalPages);

    if (clamped === this.page) {
      target.value = String(this.page);
      return;
    }

    this.dispatchEvent(
      new CustomEvent("page-change", {
        detail: { page: clamped },
        bubbles: true,
        composed: true,
      }),
    );
  }

  private handlePageFocus(e: FocusEvent) {
    (e.target as HTMLInputElement).select();
  }

  render() {
    return html`
      <div class="pagination">
        <label class="page-size">
          <span class="page-size-label">Page Size</span>
          <select
            class="input"
            .value=${String(this.size)}
            @change=${this.handleSizeChange}
          >
            ${this.sizeOptions.map(
              (n) => html`<option value=${n}>${n}</option>`,
            )}
          </select>
        </label>
        <div class="page-controls">
          <button
            class="btn btn-page"
            aria-label="Previous page"
            ?disabled=${this.page <= 1}
            @click=${this.handlePrev}
          >
            ‹
          </button>
          <span class="page-indicator">
            <input
              class="input page-input"
              type="number"
              min="1"
              max=${this.totalPages}
              step="1"
              .value=${String(this.page)}
              ?disabled=${this.totalPages <= 1}
              aria-label="Page number"
              @change=${this.handlePageInput}
              @focus=${this.handlePageFocus}
            />
            <span aria-hidden="true">/ ${this.totalPages}</span>
          </span>
          <button
            class="btn btn-page"
            aria-label="Next Page"
            ?disabled=${this.page >= this.totalPages}
            @click=${this.handleNext}
          >
            ›
          </button>
        </div>
      </div>
    `;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    "hd-pagination": PaginationControls;
  }
}
