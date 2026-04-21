# Issue #134 — Configurable page size selector for list views

## Context

During the IL6 deployment it became clear that users want to see more list results at a time without paginating repeatedly. Both `document-grid` and `prompt-list` currently hardcode `page_size: 12` in their fetch requests. This task makes the page size user-controlled.

No server changes — both `SearchRequest` types (`app/client/domains/documents/document.ts:27`, `app/client/domains/prompts/prompt.ts:40`) already accept an optional `page_size`.

## Approach

Extend the existing `<hd-pagination>` element rather than introducing a sibling element. Pagination and per-page size are one conceptual footer unit — always rendered together, emit related events, share the same divider and spacing. Merging keeps the module templates simple (one element, one footer), avoids a new CSS wrapper in both modules, and avoids relocating the existing `border-top` on `pagination-controls.module.css`.

The element gains:
- `@property() size` — current page size (default `12`)
- `@property() sizeOptions` — the selectable values (default `[12, 24, 48, 96]`)
- Emits `page-size-change` with `detail: { size: number }` (in addition to its existing `page-change`)

Each module:
- Promotes the hardcoded `12` to `@state() private pageSize = 12` (camelCase follows the existing `selectedIds` / `deleteDocument` pattern; `page_size` snake_case stays on the wire).
- Binds `.size` and handles `page-size-change` — reset `page = 1` and refetch.

## Files

| File | Change |
|------|--------|
| `app/client/ui/elements/pagination-controls.ts` | Add `size` / `sizeOptions` props, page-size handler, render size selector in footer |
| `app/client/ui/elements/pagination-controls.module.css` | Split into `.page-size` (left) + `.page-controls` (right) via `justify-content: space-between` |
| `app/client/ui/modules/document-grid.ts` | Add `pageSize` state, use in `fetchDocuments`, bind to element, handle change |
| `app/client/ui/modules/prompt-list.ts` | Same as document-grid |

Module CSS files are unchanged — the element owns its footer layout.

## Implementation

### Step 1 — Extend `app/client/ui/elements/pagination-controls.ts`

Add `inputStyles` to the styles array, add two new `@property()` fields, add the change handler, and restructure `render()` so the size selector sits on the left and the prev/indicator/next group on the right:

```typescript
import { LitElement, html } from "lit";
import { customElement, property } from "lit/decorators.js";

import buttonStyles from "@styles/buttons.module.css";
import inputStyles from "@styles/inputs.module.css";
import styles from "./pagination-controls.module.css";

@customElement("hd-pagination")
export class PaginationControls extends LitElement {
  static styles = [buttonStyles, inputStyles, styles];

  @property({ type: Number }) page = 1;
  @property({ type: Number, attribute: "total-pages" }) totalPages = 1;
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

  render() {
    return html`
      <div class="pagination">
        <label class="page-size">
          <span class="label">Per page:</span>
          <select
            class="input"
            .value=${String(this.size)}
            @change=${this.handleSizeChange}
          >
            ${this.sizeOptions.map(n => html`<option value=${n}>${n}</option>`)}
          </select>
        </label>
        <div class="page-controls">
          <button class="btn" ?disabled=${this.page <= 1} @click=${this.handlePrev}>
            Prev
          </button>
          <span class="page-indicator">
            Page ${this.page} of ${this.totalPages}
          </span>
          <button class="btn" ?disabled=${this.page >= this.totalPages} @click=${this.handleNext}>
            Next
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
```

### Step 2 — Update `app/client/ui/elements/pagination-controls.module.css`

Switch `.pagination` to `space-between` and add selectors for the new `.page-size` / `.page-controls` / `.label` pieces. Keep the existing `border-top` and `padding-top` so the divider still spans the full footer.

```css
:host {
  display: block;
}

.pagination {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
  padding-top: var(--space-3);
  border-top: 1px solid var(--divider);
}

.page-size {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.page-controls {
  display: flex;
  align-items: center;
  gap: var(--space-3);
}

.label,
.page-indicator {
  font-size: var(--text-sm);
  color: var(--color-1);
}
```

### Step 3 — Wire `app/client/ui/modules/document-grid.ts`

- Add `@state() private pageSize = 12;` alongside existing `@state()` fields.
- In `fetchDocuments()`, change `page_size: 12` → `page_size: this.pageSize`.
- Add a handler next to `handlePageChange`:

  ```typescript
  private handlePageSizeChange(e: CustomEvent<{ size: number }>) {
    this.pageSize = e.detail.size;
    this.page = 1;
    this.fetchDocuments();
  }
  ```

- Extend the existing `<hd-pagination>` render with the two new bindings:

  ```typescript
  <hd-pagination
    .page=${this.documents?.page ?? 1}
    .totalPages=${this.documents?.total_pages ?? 1}
    .size=${this.pageSize}
    @page-change=${this.handlePageChange}
    @page-size-change=${this.handlePageSizeChange}
  ></hd-pagination>
  ```

### Step 4 — Wire `app/client/ui/modules/prompt-list.ts`

Same shape as Step 3 against `fetchPrompts()` — add `@state() private pageSize = 12`, use `this.pageSize` in the request, add `handlePageSizeChange` that resets `page = 1` and calls `fetchPrompts()`, bind `.size` + `@page-size-change` on `<hd-pagination>`.

## Validation

- `mise run dev` + `bun run watch` — visit `/app/documents` and `/app/prompts`:
  - Footer shows "Per page: [12 ▼]" on the left and "Prev / Page X of Y / Next" on the right, divider spanning the full row.
  - Changing the size refetches the list and the page indicator resets to "Page 1 of N".
  - Existing search / status / sort filters still work and don't disturb `pageSize`.
  - Bulk-select / classify / delete interactions on `document-grid` still behave as before.
- `mise run vet` — no type or lint regressions.
- `mise run test` — existing suites pass.
