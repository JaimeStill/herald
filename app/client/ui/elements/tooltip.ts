import { LitElement, html } from "lit";
import { customElement, property, query } from "lit/decorators.js";

import styles from "./tooltip.module.css";

let tooltipSeq = 0;

const SHOW_DELAY_MS = 150;

/**
 * Hover- and focus-triggered tooltip primitive. Wraps a trigger via `<slot>`
 * and renders a `popover="hint"` hint anchored to it with CSS Anchor
 * Positioning (flips above/below as space allows).
 *
 * The tooltip is a dumb primitive — it always shows on hover or focus after
 * a short delay. Any gating behavior (for example, only showing when the
 * trigger content is truncated) is the composing element's responsibility.
 *
 * `popover="hint"` is deliberately chosen over `"auto"`: hints do not close
 * open auto popovers, so hovering a tooltip inside a menu leaves the menu
 * open. Hints do close other hints, so only one tooltip is ever visible.
 */
@customElement("hd-tooltip")
export class Tooltip extends LitElement {
  static styles = [styles];

  /** Text rendered inside the tooltip hint. */
  @property() message = "";

  @query("span.trigger") private triggerEl!: HTMLSpanElement;
  @query("div.tip") private tipEl!: HTMLDivElement;

  private anchorName = `--hd-tooltip-${++tooltipSeq}`;
  private showTimer: number | undefined;

  firstUpdated() {
    this.triggerEl.style.setProperty("anchor-name", this.anchorName);
    this.tipEl.style.setProperty("position-anchor", this.anchorName);
  }

  disconnectedCallback() {
    super.disconnectedCallback();
    this.clearShowTimer();
    if (this.tipEl?.matches(":popover-open")) this.tipEl.hidePopover();
  }

  private handleEnter = () => {
    this.clearShowTimer();
    this.showTimer = window.setTimeout(() => {
      if (!this.tipEl.matches(":popover-open")) this.tipEl.showPopover();
    }, SHOW_DELAY_MS);
  };

  private handleLeave = () => {
    this.clearShowTimer();
    if (this.tipEl?.matches(":popover-open")) this.tipEl.hidePopover();
  };

  private clearShowTimer() {
    if (this.showTimer !== undefined) {
      clearTimeout(this.showTimer);
      this.showTimer = undefined;
    }
  }

  render() {
    return html`
      <span
        class="trigger"
        @mouseenter=${this.handleEnter}
        @mouseleave=${this.handleLeave}
        @focusin=${this.handleEnter}
        @focusout=${this.handleLeave}
      >
        <slot></slot>
      </span>
      <div class="tip" popover="hint" role="tooltip">${this.message}</div>
    `;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    "hd-tooltip": Tooltip;
  }
}
