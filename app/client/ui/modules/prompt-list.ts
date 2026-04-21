import { LitElement, html, nothing } from "lit";
import { customElement, property, state } from "lit/decorators.js";

import type { PageResult } from "@core";
import { PromptService } from "@domains/prompts";
import type { Prompt, SearchRequest } from "@domains/prompts";

import buttonStyles from "@styles/buttons.module.css";
import inputStyles from "@styles/inputs.module.css";
import scrollStyles from "@styles/scroll.module.css";
import styles from "./prompt-list.module.css";

/**
 * Stateful module that manages the prompt browsing experience.
 * Owns search, filtering, sorting, pagination, activate/deactivate
 * lifecycle, and delete confirmation.
 */
@customElement("hd-prompt-list")
export class PromptList extends LitElement {
  static styles = [buttonStyles, inputStyles, scrollStyles, styles];

  @property({ type: Object }) selected: Prompt | null = null;

  @state() private prompts: PageResult<Prompt> | null = null;
  @state() private page = 1;
  @state() private search = "";
  @state() private stage = "";
  @state() private sort = "Name";
  @state() private deletePrompt: Prompt | null = null;

  private searchTimer = 0;

  connectedCallback() {
    super.connectedCallback();
    this.fetchPrompts();
  }

  disconnectedCallback() {
    super.disconnectedCallback();
    clearTimeout(this.searchTimer);
  }

  async refresh() {
    this.page = 1;
    await this.fetchPrompts();
  }

  private async fetchPrompts() {
    const req: SearchRequest = {
      page: this.page,
      page_size: 12,
      sort: this.sort,
    };

    if (this.search) req.search = this.search;
    if (this.stage) req.stage = this.stage as SearchRequest["stage"];

    const result = await PromptService.search(req);

    if (result.ok) this.prompts = result.data;
  }

  private handleSearchInput(e: Event) {
    const input = e.target as HTMLInputElement;
    this.search = input.value;

    clearTimeout(this.searchTimer);
    this.searchTimer = window.setTimeout(() => this.refresh(), 300);
  }

  private handleStageFilter(e: Event) {
    const select = e.target as HTMLSelectElement;
    this.stage = select.value;
    this.refresh();
  }

  private handleSort(e: Event) {
    const select = e.target as HTMLSelectElement;
    this.sort = select.value;
    this.refresh();
  }

  private handlePageChange(e: CustomEvent<{ page: number }>) {
    this.page = e.detail.page;
    this.fetchPrompts();
  }

  private handleSelect(e: CustomEvent<{ prompt: Prompt }>) {
    this.dispatchEvent(
      new CustomEvent("select", {
        detail: { prompt: e.detail.prompt },
        bubbles: true,
        composed: true,
      }),
    );
  }

  private async handleToggleActive(
    e: CustomEvent<{ id: string; active: boolean }>,
  ) {
    const { id, active } = e.detail;
    const result = active
      ? await PromptService.activate(id)
      : await PromptService.deactivate(id);

    if (result.ok) this.fetchPrompts();
  }

  private handleDelete(e: CustomEvent<{ prompt: Prompt }>) {
    this.deletePrompt = e.detail.prompt;
  }

  private async confirmDelete() {
    if (!this.deletePrompt) return;

    const id = this.deletePrompt.id;
    this.deletePrompt = null;

    const result = await PromptService.delete(id);

    if (result.ok) {
      this.dispatchEvent(
        new CustomEvent("delete", {
          detail: { id },
          bubbles: true,
          composed: true,
        }),
      );
      this.fetchPrompts();
    }
  }

  private cancelDelete() {
    this.deletePrompt = null;
  }

  private handleNew() {
    this.dispatchEvent(
      new CustomEvent("create", {
        bubbles: true,
        composed: true,
      }),
    );
  }

  private renderToolbar() {
    return html`
      <div class="toolbar">
        <input
          type="search"
          class="input search-input"
          placeholder="Search prompts..."
          .value=${this.search}
          @input=${this.handleSearchInput}
        />
        <button class="btn btn-blue" @click=${this.handleNew}>New</button>
        <select class="input filter-select" @change=${this.handleStageFilter}>
          <option value="">---</option>
          <option value="classify" ?selected=${this.stage === "classify"}>
            Classify
          </option>
          <option value="enhance" ?selected=${this.stage === "enhance"}>
            Enhance
          </option>
          <option value="finalize" ?selected=${this.stage === "finalize"}>
            Finalize
          </option>
        </select>
        <select class="input sort-select" @change=${this.handleSort}>
          <option value="Name" ?selected=${this.sort === "Name"}>
            Name (A-Z)
          </option>
          <option value="-Name" ?selected=${this.sort === "-Name"}>
            Name (Z-A)
          </option>
          <option value="Stage" ?selected=${this.sort === "Stage"}>
            Stage (A-Z)
          </option>
          <option value="-Stage" ?selected=${this.sort === "-Stage"}>
            Stage (Z-A)
          </option>
        </select>
      </div>
    `;
  }

  private renderList() {
    if (!this.prompts) {
      return html`<div class="empty-state">Loading...</div>`;
    }

    if (this.prompts.data.length < 1) {
      return html`<div class="empty-state">No prompts found.</div>`;
    }

    return html`
      <div class="list scroll-y">
        ${this.prompts.data.map(
          (prompt) => html`
            <hd-prompt-card
              .prompt=${prompt}
              ?selected=${this.selected?.id === prompt.id}
              @select=${this.handleSelect}
              @toggle-active=${this.handleToggleActive}
              @delete=${this.handleDelete}
            ></hd-prompt-card>
          `,
        )}
      </div>
    `;
  }

  render() {
    const message = `Are you sure you want to delete ${this.deletePrompt?.name}?`;

    return html`
      ${this.renderToolbar()} ${this.renderList()}
      <hd-pagination
        .page=${this.prompts?.page ?? 1}
        .totalPages=${this.prompts?.total_pages ?? 1}
        @page-change=${this.handlePageChange}
      ></hd-pagination>
      ${this.deletePrompt
        ? html`
            <hd-confirm-dialog
              message=${message}
              @confirm=${this.confirmDelete}
              @cancel=${this.cancelDelete}
            ></hd-confirm-dialog>
          `
        : nothing}
    `;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    "hd-prompt-list": PromptList;
  }
}
