import { LitElement, html } from "lit";
import { customElement, property, query } from "lit/decorators.js";

import buttonStyles from "@styles/buttons.module.css";
import styles from "./confirm-dialog.module.css";

/**
 * Semantic variant of the confirm button. Maps to a button color class:
 * `danger` → red (destructive), `primary` → green (affirmative),
 * `neutral` → default. The default is `danger` since confirmation dialogs
 * most commonly guard destructive actions.
 */
export type ConfirmKind = "danger" | "primary" | "neutral";

const CONFIRM_CLASS: Record<ConfirmKind, string> = {
  danger: "btn btn-red",
  primary: "btn btn-green",
  neutral: "btn",
};

/**
 * Modal confirmation dialog rendered via `<dialog>.showModal()`. The parent
 * conditionally mounts this element when it needs to confirm an action and
 * listens for `confirm` / `cancel` custom events.
 *
 * `.showModal()` provides focus trap, Escape-to-cancel, focus return on close,
 * and `::backdrop` styling natively — no manual scrim, no `z-index`.
 */
@customElement("hd-confirm-dialog")
export class ConfirmDialog extends LitElement {
  static styles = [buttonStyles, styles];

  /** Prompt text shown to the user. */
  @property() message = "Are you sure?";
  /** Semantic variant controlling the Confirm button color. Defaults to `danger`. */
  @property() confirmKind: ConfirmKind = "danger";

  @query("dialog") private dialogEl!: HTMLDialogElement;

  firstUpdated() {
    this.dialogEl.showModal();
  }

  private handleConfirm() {
    this.dispatchEvent(
      new CustomEvent("confirm", { bubbles: true, composed: true }),
    );
    this.dialogEl.close();
  }

  private handleCancel() {
    this.dispatchEvent(
      new CustomEvent("cancel", { bubbles: true, composed: true }),
    );
  }

  private handleBackdropClick(e: MouseEvent) {
    if (e.target === this.dialogEl) this.handleCancel();
  }

  private handleCancelEvent(e: Event) {
    e.preventDefault();
    this.handleCancel();
  }

  render() {
    return html`
      <dialog
        @click=${this.handleBackdropClick}
        @cancel=${this.handleCancelEvent}
      >
        <div class="panel">
          <p class="message">${this.message}</p>
          <div class="actions">
            <button class="btn" @click=${this.handleCancel}>Cancel</button>
            <button
              class=${CONFIRM_CLASS[this.confirmKind]}
              @click=${this.handleConfirm}
              autofocus
            >
              Confirm
            </button>
          </div>
        </div>
      </dialog>
    `;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    "hd-confirm-dialog": ConfirmDialog;
  }
}
