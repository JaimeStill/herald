# 135 - URL query-parameter state persistence for list views

## Problem Context

List-view state (pagination, search, filter, sort) on `hd-document-grid` and `hd-prompt-list` lives only in `@state` and is destroyed on view unmount. Clicking into `/review/:id` and returning to `/`, or reloading the page, drops the user back at defaults.

This task adds two-way URL binding: reads on mount, writes on each filter change. Writes must use `history.replaceState` — `pushState` plus the router's `innerHTML = ""` teardown (router.ts:108) would remount the view on every filter change and destroy the state we're preserving.

## Architecture Approach

- Keep filter fields as `@state` (internal state), not `@property`. The codebase reserves `@property` for parent-passed inputs.
- Retire the router's unused query-attribute splat (router.ts:115-117) and replace it with explicit `queryParams()` / `updateQuery()` helpers views call on demand.
- `updateQuery` is built on `URLSearchParams` + `history.replaceState`. Do not reuse `toQueryString` — that helper is coupled to API request URL construction.
- Per-view `DEFAULTS` const centralizes the canonical default values; field initializers, hydration fallbacks, and sync-omission checks all reference it.

## Implementation

### Step 1: Add `queryParams` and `updateQuery` helpers; remove query-attribute splat

**File:** `app/client/core/router/router.ts`

In `mount()`, delete the query-attribute splat at lines 115-117:

```ts
// REMOVE this block:
for (const [key, value] of Object.entries(match.query)) {
  el.setAttribute(key, value);
}
```

(Path-param splatting at 111-113 stays — it's in use by `hd-review-view` and `hd-not-found-view`.)

Append two exported functions at the top of the file, near the existing `navigate` export:

```ts
export function queryParams(): Record<string, string> {
  const params = new URLSearchParams(location.search);
  const result: Record<string, string> = {};
  for (const [key, value] of params) result[key] = value;
  return result;
}

export function updateQuery(
  patch: Record<string, string | number | undefined | null>,
): void {
  const url = new URL(location.href);
  for (const [key, value] of Object.entries(patch)) {
    if (value === undefined || value === null || value === "") {
      url.searchParams.delete(key);
    } else {
      url.searchParams.set(key, String(value));
    }
  }
  history.replaceState(null, "", url.pathname + url.search);
}
```

### Step 2: Re-export the helpers from the router barrel

**File:** `app/client/core/router/index.ts`

Change:

```ts
export { Router, navigate } from "./router";
```

to:

```ts
export { Router, navigate, queryParams, updateQuery } from "./router";
```

### Step 3: Wire URL state into `hd-document-grid`

**File:** `app/client/ui/modules/document-grid.ts`

Update the router import to include the new helpers:

```ts
import { navigate, queryParams, updateQuery } from "@core/router";
```

Add a `DEFAULTS` const immediately below the imports (before the `ClassifyProgress` interface):

```ts
const DEFAULTS = {
  page: 1,
  pageSize: 12,
  search: "",
  status: "",
  sort: "-UploadedAt",
} as const;
```

Replace the `@state` initializers for the filter fields to reference `DEFAULTS`:

```ts
@state() private page = DEFAULTS.page;
@state() private pageSize = DEFAULTS.pageSize;
@state() private search = DEFAULTS.search;
@state() private status = DEFAULTS.status;
@state() private sort = DEFAULTS.sort;
```

Leave `documents`, `classifying`, `selectedIds`, `deleteDocument` unchanged.

Update `connectedCallback` to hydrate before fetching:

```ts
connectedCallback() {
  super.connectedCallback();
  this.hydrateFromQuery();
  this.fetchDocuments();
}
```

Add two new private methods (place them next to `fetchDocuments`):

```ts
private hydrateFromQuery() {
  const q = queryParams();
  if (q.page) this.page = Number(q.page) || DEFAULTS.page;
  if (q.page_size) this.pageSize = Number(q.page_size) || DEFAULTS.pageSize;
  if (q.search) this.search = q.search;
  if (q.status) this.status = q.status;
  if (q.sort) this.sort = q.sort;
}

private syncQuery() {
  updateQuery({
    page: this.page === DEFAULTS.page ? undefined : this.page,
    page_size: this.pageSize === DEFAULTS.pageSize ? undefined : this.pageSize,
    search: this.search || undefined,
    status: this.status || undefined,
    sort: this.sort === DEFAULTS.sort ? undefined : this.sort,
  });
}
```

Fold `syncQuery` into `refresh`:

```ts
async refresh() {
  this.page = DEFAULTS.page;
  this.syncQuery();
  await this.fetchDocuments();
}
```

Update the page-level handlers to sync the URL before fetching:

```ts
private handlePageChange(e: CustomEvent<{ page: number }>) {
  this.page = e.detail.page;
  this.syncQuery();
  this.fetchDocuments();
}

private handlePageSizeChange(e: CustomEvent<{ size: number }>) {
  this.pageSize = e.detail.size;
  this.page = DEFAULTS.page;
  this.syncQuery();
  this.fetchDocuments();
}
```

`handleSearchInput`, `handleStatusFilter`, and `handleSort` already funnel through `refresh()` and need no further changes.

### Step 4: Wire URL state into `hd-prompt-list`

**File:** `app/client/ui/modules/prompt-list.ts`

Add the router import (this module does not currently import from `@core/router`):

```ts
import { queryParams, updateQuery } from "@core/router";
```

Add a `DEFAULTS` const below the imports:

```ts
const DEFAULTS = {
  page: 1,
  pageSize: 12,
  search: "",
  stage: "",
  sort: "Name",
} as const;
```

Replace the filter `@state` initializers:

```ts
@state() private page = DEFAULTS.page;
@state() private pageSize = DEFAULTS.pageSize;
@state() private search = DEFAULTS.search;
@state() private stage = DEFAULTS.stage;
@state() private sort = DEFAULTS.sort;
```

Leave `prompts` and `deletePrompt` unchanged. `selected` is already `@property` and stays.

Update `connectedCallback`:

```ts
connectedCallback() {
  super.connectedCallback();
  this.hydrateFromQuery();
  this.fetchPrompts();
}
```

Add the two new private methods next to `fetchPrompts`:

```ts
private hydrateFromQuery() {
  const q = queryParams();
  if (q.page) this.page = Number(q.page) || DEFAULTS.page;
  if (q.page_size) this.pageSize = Number(q.page_size) || DEFAULTS.pageSize;
  if (q.search) this.search = q.search;
  if (q.stage) this.stage = q.stage;
  if (q.sort) this.sort = q.sort;
}

private syncQuery() {
  updateQuery({
    page: this.page === DEFAULTS.page ? undefined : this.page,
    page_size: this.pageSize === DEFAULTS.pageSize ? undefined : this.pageSize,
    search: this.search || undefined,
    stage: this.stage || undefined,
    sort: this.sort === DEFAULTS.sort ? undefined : this.sort,
  });
}
```

Fold `syncQuery` into `refresh`:

```ts
async refresh() {
  this.page = DEFAULTS.page;
  this.syncQuery();
  await this.fetchPrompts();
}
```

Update the page-level handlers:

```ts
private handlePageChange(e: CustomEvent<{ page: number }>) {
  this.page = e.detail.page;
  this.syncQuery();
  this.fetchPrompts();
}

private handlePageSizeChange(e: CustomEvent<{ size: number }>) {
  this.pageSize = e.detail.size;
  this.page = DEFAULTS.page;
  this.syncQuery();
  this.fetchPrompts();
}
```

`handleSearchInput`, `handleStageFilter`, and `handleSort` already funnel through `refresh()` and need no further changes.

## Remediation

### R1: Pagination controls don't scale into narrow containers

`hd-pagination` was designed against the full-width documents toolbar. When embedded in the narrow prompt-list column (~300px), the labels wrap and the layout breaks. Rather than maintaining two layouts conditionally, simplify the component permanently: drop the textual labels ("Per page:", "Prev", "Page", "of N", "Next") in favor of a compact form that works at any width — `[size-select]  ‹  [page-input]/N  ›`. Nothing is lost functionally, the per-page select remains a dropdown, and a11y is preserved by moving the textual labels to `aria-label` attributes.

#### Markup changes

**File:** `app/client/ui/elements/pagination-controls.ts`

Replace the `render()` method body with the simplified form. `label` element becomes a plain wrapper (no longer pairing a visible caption with the select), and the prev/next buttons use chevron glyphs with accessible labels:

```ts
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
          aria-label="Next page"
          ?disabled=${this.page >= this.totalPages}
          @click=${this.handleNext}
        >
          ›
        </button>
      </div>
    </div>
  `;
}
```

The `<label class="page-size">` pairs the visible caption with the select via native label association. Keeping the `aria-label` off the select (native label is preferred when present).

#### Style changes

**File:** `app/client/ui/elements/pagination-controls.module.css`

Keep `.page-size` (now wrapping the label + select again), add a `.page-size-label` rule, and make the chevron buttons stretch to match the height of their sibling `.input` controls so everything is flush. Chevron glyphs get a larger font-size (`--text-xl`) so they're clearly readable:

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

.page-size-label {
  font-size: var(--text-sm);
  color: var(--color-1);
}

.page-controls {
  display: flex;
  align-items: center;
  gap: var(--space-2);
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
  appearance: textfield;
}

.page-input::-webkit-inner-spin-button,
.page-input::-webkit-outer-spin-button {
  appearance: none;
  margin: 0;
}

.btn-page {
  align-self: stretch;
  min-width: auto;
  padding: 0 var(--space-3);
  font-size: var(--text-xl);
  line-height: 1;
}
```

`align-self: stretch` on `.btn-page` overrides the flex container's `align-items: center` so the chevron buttons fill the cross-axis height of the flex line — which is determined by the tallest sibling (the page-size select and the page input). No hardcoded heights; the buttons auto-match regardless of theme/density changes.

The `appearance: textfield` on `.page-input` plus the `::-webkit-inner/outer-spin-button` rules remove the native number-input spinner arrows and the reserved space they occupy. `width: auto` is retained intentionally — the input renders at its intrinsic content width, so it grows naturally as page counts scale into 4-5+ digits (relevant for the ~750k-1M document corpus).

## Validation Criteria

- [ ] On `/documents`, changing search / status / sort / page / page_size updates the URL with only non-default values and re-renders the grid.
- [ ] Browser history length stays constant across filter changes (`replaceState`, not `pushState`).
- [ ] Navigating `/documents` → `/review/:id` → Back restores filter state exactly.
- [ ] Hard-reloading `/documents?page=2&status=review&sort=Filename` mounts with those values and fires a single fetch.
- [ ] Same behaviors verified on `/prompts` with `stage` filter and default sort `"Name"`.
- [ ] Default state (no filter touched) leaves the URL at `/documents` or `/prompts` with no `?`.
- [ ] `bun run build` succeeds with no type errors.
- [ ] `mise run vet` succeeds with no regressions.
- [ ] Simplified `hd-pagination` renders cleanly in both the wide documents toolbar and the narrow prompt-list column; chevron buttons and `[input]/N` indicator fit without wrapping; per-page select remains functional.
