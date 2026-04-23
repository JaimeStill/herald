import { LitElement, html, nothing } from "lit";
import { customElement, property } from "lit/decorators.js";

import type { Prompt } from "@domains/prompts";
import "./tooltip";

import badgeStyles from "@styles/badge.module.css";
import buttonStyles from "@styles/buttons.module.css";
import cardStyles from "@styles/cards.module.css";
import styles from "./prompt-card.module.css";

/**
 * Pure element that displays a prompt's name, stage badge, active indicator,
 * and description. Dispatches `select`, `toggle-active`, and `delete` custom events.
 */
@customElement("hd-prompt-card")
export class PromptCard extends LitElement {
  static styles = [buttonStyles, badgeStyles, cardStyles, styles];

  @property({ type: Object }) prompt!: Prompt;
  @property({ type: Boolean }) selected = false;

  private handleSelect() {
    this.dispatchEvent(
      new CustomEvent("select", {
        detail: { prompt: this.prompt },
        bubbles: true,
        composed: true,
      }),
    );
  }

  private handleToggleActive() {
    this.dispatchEvent(
      new CustomEvent("toggle-active", {
        detail: { id: this.prompt.id, active: !this.prompt.active },
        bubbles: true,
        composed: true,
      }),
    );
  }

  private handleDelete() {
    this.dispatchEvent(
      new CustomEvent("delete", {
        detail: { prompt: this.prompt },
        bubbles: true,
        composed: true,
      }),
    );
  }

  private renderDescription() {
    if (!this.prompt.description) return nothing;

    return html` <div class="description">${this.prompt.description}</div> `;
  }

  render() {
    const p = this.prompt;

    return html`
      <div class="card ${this.selected ? "selected" : ""}">
        <div class="header" @click=${this.handleSelect}>
          <hd-tooltip .message=${p.name}>
            <span class="name">${p.name}</span>
          </hd-tooltip>
          <span class="badge ${p.stage}">${p.stage}</span>
          <span
            class="active-indicator ${p.active ? "active" : ""}"
            title=${p.active ? "Active" : "Inactive"}
          ></span>
        </div>

        ${this.renderDescription()}

        <div class="actions">
          <button
            class="btn ${p.active ? "btn-yellow" : "btn-green"}"
            @click=${this.handleToggleActive}
          >
            ${p.active ? "Deactivate" : "Activate"}
          </button>
          <button class="btn btn-red" @click=${this.handleDelete}>
            Delete
          </button>
        </div>
      </div>
    `;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    "hd-prompt-card": PromptCard;
  }
}
