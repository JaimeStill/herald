import { LitElement, html, nothing } from "lit";
import { customElement, state } from "lit/decorators.js";

import type { Prompt } from "@domains/prompts";

import styles from "./prompts-view.module.css";

/** Route-level view for prompt management. */
@customElement("hd-prompts-view")
export class PromptsView extends LitElement {
  static styles = styles;

  @state() private selectedPrompt: Prompt | null = null;
  @state() private showForm = false;

  private reset() {
    this.showForm = false;
    this.selectedPrompt = null;
  }

  private handleCreate() {
    this.selectedPrompt = null;
    this.showForm = true;
  }

  private async handlePromptSelect(e: CustomEvent<{ prompt: Prompt }>) {
    this.selectedPrompt = e.detail.prompt;
    this.showForm = true;
  }

  private handlePromptSaved() {
    this.reset();
    this.renderRoot.querySelector<any>("hd-prompt-list").refresh();
  }

  private handleCancel() {
    this.reset();
  }

  private handlePromptDeleted(e: CustomEvent<{ id: string }>) {
    if (this.selectedPrompt?.id === e.detail.id) {
      this.reset();
    }
  }

  render() {
    return html`
      <div class="view">
        <div class="view-header">
          <h1>Prompts</h1>
        </div>
        <div class="view-content">
          <div class="list-panel">
            <hd-prompt-list
              .selected=${this.selectedPrompt}
              @create=${this.handleCreate}
              @select=${this.handlePromptSelect}
              @delete=${this.handlePromptDeleted}
            ></hd-prompt-list>
          </div>
          ${this.showForm
            ? html`
                <div class="form-panel">
                  <hd-prompt-form
                    .prompt=${this.selectedPrompt}
                    @save=${this.handlePromptSaved}
                    @cancel=${this.handleCancel}
                  ></hd-prompt-form>
                </div>
              `
            : nothing}
        </div>
      </div>
    `;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    "hd-prompts-view": PromptsView;
  }
}
