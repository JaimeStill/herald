# 134 - Configurable page size selector for list views

## Problem Context

Both `document-grid` and `prompt-list` hardcode `page_size: 12` in their fetch requests. During IL6 usage it became clear users want to see more results at once without paginating repeatedly. Both `SearchRequest` types already accept an optional `page_size`, so this is a pure client-side change.

## Architecture Approach

Extend the existing `<hd-pagination>` element rather than adding a sibling element. Pagination and per-page size are one conceptual footer unit — always rendered together, sharing the same divider and spacing. Merging keeps the module templates simple (one element, one footer), avoids a new CSS wrapper in both modules, and keeps the existing `border-top` on `pagination-controls.module.css` in place.

The element gains `size` / `sizeOptions` properties and a `page-size-change` event in addition to its existing `page-change`. Each module promotes the hardcoded `12` to a `pageSize` `@state()` field, binds it to the element, and on change resets `page = 1` and refetches.

The static "Page X of N" indicator also becomes an editable `<input type="number">` so users can jump directly to a specific page. The input commits on `change` (blur / Enter), not `input`, so refetches don't fire per keystroke. Values are `Math.trunc`'d then clamped to `[1, totalPages]`; invalid or out-of-range entries are silently corrected and the DOM value is rebound to `this.page`. Browser-native spinner arrows and mouse-wheel stepping are left as-is (no overrides). The input is disabled when `totalPages <= 1`.

## Implementation

### Step 1 — Extend `app/client/ui/elements/pagination-controls.ts`

Replace the existing file with the full implementation below. Changes from today's file: add `inputStyles`, add `size` / `sizeOptions` `@property()` fields, add `handleSizeChange` + `handlePageInput` + `handlePageFocus` methods, and restructure `render()` so the footer has a left `.page-size` label and a right `.page-controls` group with an editable page-number input replacing the static indicator.

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

    const clamped = Math.min(
      Math.max(Math.trunc(raw), 1),
      this.totalPages,
    );

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
          <span class="label">Per page:</span>
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
            class="btn"
            ?disabled=${this.page <= 1}
            @click=${this.handlePrev}
          >
            Prev
          </button>
          <span class="page-indicator">
            Page
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
            of ${this.totalPages}
          </span>
          <button
            class="btn"
            ?disabled=${this.page >= this.totalPages}
            @click=${this.handleNext}
          >
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

Replace the file with the following. Changes: `.pagination` flips from `justify-content: center` to `justify-content: space-between`; new rules add `.page-size`, `.page-controls`, `.label`, and `.page-input`. `.page-indicator` becomes a flex row so its "Page" / `<input>` / "of N" children align on the input's baseline without vertical jitter.

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

.label {
  font-size: var(--text-sm);
  color: var(--color-1);
}

.page-indicator {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  font-size: var(--text-sm);
  color: var(--color-1);
}

.page-input {
  width: auto;
  padding: var(--space-2) var(--space-1);
  text-align: center;
}
```

### Step 3 — Wire `app/client/ui/modules/document-grid.ts`

Three incremental changes.

**3a.** Add `pageSize` alongside the existing `@state()` fields (the block around line 30):

```typescript
@state() private pageSize = 12;
```

**3b.** In `fetchDocuments()`, replace the hardcoded page size:

```typescript
const req: SearchRequest = {
  page: this.page,
  page_size: this.pageSize,
  sort: this.sort,
};
```

**3c.** Add the handler next to `handlePageChange`:

```typescript
private handlePageSizeChange(e: CustomEvent<{ size: number }>) {
  this.pageSize = e.detail.size;
  this.page = 1;
  this.fetchDocuments();
}
```

**3d.** Extend the existing `<hd-pagination>` render with two new bindings:

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

Mirror Step 3 against `fetchPrompts()` and the prompt-list's `<hd-pagination>` render.

**4a.** Add `@state() private pageSize = 12;` alongside the existing `@state()` fields.

**4b.** In `fetchPrompts()`:

```typescript
const req: SearchRequest = {
  page: this.page,
  page_size: this.pageSize,
  sort: this.sort,
};
```

**4c.** Add the handler next to `handlePageChange`:

```typescript
private handlePageSizeChange(e: CustomEvent<{ size: number }>) {
  this.pageSize = e.detail.size;
  this.page = 1;
  this.fetchPrompts();
}
```

**4d.** Extend the existing `<hd-pagination>` render:

```typescript
<hd-pagination
  .page=${this.prompts?.page ?? 1}
  .totalPages=${this.prompts?.total_pages ?? 1}
  .size=${this.pageSize}
  @page-change=${this.handlePageChange}
  @page-size-change=${this.handlePageSizeChange}
></hd-pagination>
```

## Validation Criteria

- [ ] `<hd-pagination>` footer renders "Per page: [12 ▼]" on the left and "Prev / Page [n] of N / Next" on the right, divider spanning the full row.
- [ ] Options list is `[12, 24, 48, 96]`, current selection matches the module's `pageSize`.
- [ ] Changing the selector refetches the list (`document-grid` and `prompt-list`) and resets `page` to 1.
- [ ] `SearchRequest.page_size` on the network shows the chosen value (check devtools / server logs).
- [ ] Page input: typing a valid page and blurring / pressing Enter navigates to that page.
- [ ] Page input: typing `0`, a negative, or a value above `totalPages` snaps to the nearest valid value; a decimal truncates (`2.9` → `2`); non-numeric / empty reverts to the current page without navigating.
- [ ] Page input: focusing the input auto-selects its contents so typing immediately overwrites.
- [ ] Page input: spinner arrows and mouse-wheel stepping work and are constrained by `min` / `max`.
- [ ] Page input: when `totalPages <= 1`, the input is disabled and shows `1`.
- [ ] Existing search / status / sort / stage filters still work and do not disturb `pageSize`.
- [ ] Bulk-select / classify / delete interactions on `document-grid` still behave as before.
- [ ] `mise run vet` passes with no new warnings.
- [ ] `mise run test` passes.
