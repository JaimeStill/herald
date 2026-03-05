import { LitElement, html } from "lit";
import { customElement, property } from "lit/decorators.js";

import buttonStyles from "@styles/buttons.module.css";
import styles from "./confirm-dialog.module.css";

/**
 * Pure element that renders a confirmation dialog overlay.
 * Dispatches `confirm` when accepted and `cancel` when dismissed.
 */
@customElement("hd-confirm-dialog")
export class ConfirmDialog extends LitElement {
  static styles = [buttonStyles, styles];

  @property() message = "Are you sure?";

  private handleConfirm() {
    this.dispatchEvent(
      new CustomEvent("confirm", { bubbles: true, composed: true }),
    );
  }

  private handleCancel() {
    this.dispatchEvent(
      new CustomEvent("cancel", { bubbles: true, composed: true }),
    );
  }

  render() {
    return html`
      <div class="overlay" @click=${this.handleCancel}>
        <div class="dialog" @click=${(e: Event) => e.stopPropagation()}>
          <p class="message">${this.message}</p>
          <div class="actions">
            <button class="btn" @click=${this.handleCancel}>Cancel</button>
            <button class="btn confirm-btn" @click=${this.handleConfirm}>
              Confirm
            </button>
          </div>
        </div>
      </div>
    `;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    "hd-confirm-dialog": ConfirmDialog;
  }
}
