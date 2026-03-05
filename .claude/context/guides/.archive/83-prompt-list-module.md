# 83 - Prompt List Module with Stage Filtering and Activation

## Problem Context

Second sub-issue of Objective #60 (Prompt Management View). Creates the stateful list module that manages browsing, filtering, pagination, and prompt lifecycle actions (activate/deactivate/delete). The prompt card element (#82) and prompts domain layer (types + service) are already complete.

## Architecture Approach

Follow the `hd-document-grid` module pattern. The prompt list is simpler — no SSE streaming, no bulk selection. Uses `PromptService.search()` (POST with `SearchRequest` body) for server-side filtering. Vertical list layout instead of grid (prompts are text-heavy per objective decision).

## Implementation

### Step 1: Create prompt-list.module.css

**New file:** `app/client/ui/modules/prompt-list.module.css`

```css
:host {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
  flex: 1;
  min-height: 0;
}

.toolbar {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  flex-shrink: 0;
  flex-wrap: wrap;
}

.search-input,
.filter-select,
.sort-select {
  padding: var(--space-2) var(--space-3);
  border: 1px solid var(--divider);
  border-radius: var(--radius-sm);
  background: var(--bg-1);
  color: var(--color);
  font-size: var(--text-sm);
  font-family: var(--font-mono);

  &:focus-visible {
    outline: 2px solid var(--blue);
    outline-offset: 2px;
  }
}

.search-input {
  flex: 1;
  min-width: 12rem;
}

.list {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding-bottom: var(--space-2);
}

.empty-state {
  display: flex;
  align-items: center;
  justify-content: center;
  flex: 1;
  font-size: var(--text-sm);
  color: var(--color-2);
}

.new-btn:not(:disabled) {
  border-color: var(--blue);
  color: var(--blue);

  &:hover {
    background: var(--blue-bg);
  }
}
```

### Step 2: Create prompt-list.ts

**New file:** `app/client/ui/modules/prompt-list.ts`

```typescript
import { LitElement, html, nothing } from "lit";
import { customElement, property, state } from "lit/decorators.js";

import type { PageResult } from "@core";
import { PromptService } from "@domains/prompts";
import type { Prompt, SearchRequest } from "@domains/prompts";

import buttonStyles from "@styles/buttons.module.css";
import styles from "./prompt-list.module.css";

@customElement("hd-prompt-list")
export class PromptList extends LitElement {
  static styles = [buttonStyles, styles];

  @property({ type: String }) selectedId = "";

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

  private handleSelect(e: CustomEvent<{ id: string }>) {
    this.dispatchEvent(
      new CustomEvent("prompt-select", {
        detail: { id: e.detail.id },
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
        new CustomEvent("prompt-deleted", {
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
          class="search-input"
          placeholder="Search prompts..."
          .value=${this.search}
          @input=${this.handleSearchInput}
        />
        <select class="filter-select" @change=${this.handleStageFilter}>
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
        <select class="sort-select" @change=${this.handleSort}>
          <option value="Name" ?selected=${this.sort === "Name"}>
            Name (A-Z)
          </option>
          <option value="-Name" ?selected=${this.sort === "-Name"}>
            Name (Z-A)
          </option>
          <option value="Stage" ?selected=${this.sort === "Stage"}>
            Stage
          </option>
        </select>
        <button class="btn new-btn" @click=${this.handleNew}>New</button>
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
      <div class="list">
        ${this.prompts.data.map(
          (prompt) => html`
            <hd-prompt-card
              .prompt=${prompt}
              ?selected=${this.selectedId === prompt.id}
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
              message="Are you sure you want to delete ${this.deletePrompt.name}?"
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
```

### Step 3: Update barrel export

**Modify:** `app/client/ui/modules/index.ts`

Add after existing exports:

```typescript
export { PromptList } from "./prompt-list";
```

## Validation Criteria

- [ ] Module fetches and displays paginated prompts on mount
- [ ] Search input filters with 300ms debounce
- [ ] Stage dropdown filters by workflow stage
- [ ] Toggle active calls activate/deactivate API and refreshes list
- [ ] Delete shows confirmation dialog and removes prompt on confirm
- [ ] `prompt-select`, `create`, and `prompt-deleted` events dispatched correctly
- [ ] Pagination controls navigate pages
- [ ] Public `refresh()` method works
- [ ] Barrel exports updated
- [ ] Bun build succeeds
