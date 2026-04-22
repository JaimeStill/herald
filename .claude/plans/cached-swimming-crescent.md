# Plan — Issue #135: URL query-parameter state persistence for list views

## Context

List-view state (pagination, search, filter, sort) on `hd-document-grid` and `hd-prompt-list` lives only in `@state` and is destroyed on view unmount. Clicking into `/review/:id` and returning to `/` drops the user back at defaults. Reload does the same.

This needs a two-way URL binding: reads on mount, writes on change. The egress path (view → URL) must use `history.replaceState` — `pushState` plus the router's `innerHTML = ""` teardown at router.ts:108 would remount the view on every filter change and destroy the very state we're trying to preserve.

## Design choices

### 1. Don't convert `@state` to `@property`

The existing codebase uses `@property` exclusively for **parent-passed inputs** (`document`, `documentId`, `prompt`, `selected`, `page` on `hd-pagination`, etc.). `page`/`search`/`status`/`sort`/`pageSize` on the list modules are **internal filter state** — nothing outside the module writes them. Declaring them `@property` blurs that boundary and overloads the decorator semantics.

Keep them as `@state`. Hydrate from the URL explicitly in `connectedCallback`. This is also minimally invasive — every existing handler, `refresh()`, and `fetchDocuments()` keeps its current shape.

### 2. Retire the router's query-attribute splat

Router.ts:115-117 currently does `el.setAttribute(key, value)` for each query-string entry. No view reads these, and per Design Choice 1 we don't want views reading them as attributes. Remove this block and replace it with an explicit `queryParams()` accessor that views call on demand. Path-param splatting (router.ts:111-113) stays — it's used by `hd-review-view` and `hd-not-found-view`.

### 3. Don't reuse `toQueryString`

`toQueryString` (api.ts:164) builds API request query strings: prepends `?`, takes a flat params object, always emits a complete string. Used by `documents/service.ts`, `prompts/service.ts`, `classifications/service.ts`.

URL state sync has different semantics: **merge-and-replace** on `location.search`, delete keys on empty, operate on an existing URL. Build `updateQuery` directly on `URLSearchParams` + `history.replaceState`. The two helpers stay decoupled — they solve different problems.

### 4. Centralize defaults per view

Hardcoding `page=1`, `pageSize=12`, `sort="-UploadedAt"` in the field declaration, again in the hydration fallback, and a third time in the sync-omission check is brittle. One `DEFAULTS` const at module top, referenced everywhere:

```ts
const DEFAULTS = {
  page: 1,
  pageSize: 12,
  search: "",
  status: "",
  sort: "-UploadedAt",
} as const;
```

`@state` fields initialize from `DEFAULTS.*`, hydration falls back to `DEFAULTS.*`, syncQuery omits when `this.x === DEFAULTS.x`. Scoped per view because filter fields diverge (`status` vs `stage`, different sort defaults). App-wide defaults would require server-injected config — noted as out of scope in this session; the per-view const is the natural seam for that future plumbing.

## Implementation

### 1. Router helpers

**File:** `app/client/core/router/router.ts`

Add two exports:

```ts
/** Reads the current URL's query string as a flat map. */
export function queryParams(): Record<string, string> {
  const params = new URLSearchParams(location.search);
  const result: Record<string, string> = {};
  for (const [key, value] of params) result[key] = value;
  return result;
}

/**
 * Merges a patch into the current URL's query string and replaces history.
 * Keys with undefined/null/"" values are deleted, not retained.
 * Uses replaceState (not pushState) so filter changes don't stack history
 * entries and don't trigger a view remount.
 */
export function updateQuery(patch: Record<string, string | number | undefined | null>): void {
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

Remove router.ts:115-117 (the query-attribute splat):

```ts
// DELETE:
for (const [key, value] of Object.entries(match.query)) {
  el.setAttribute(key, value);
}
```

`RouteMatch.query` stays on the type for now — it's still populated by `match()` and cheap to keep for future use. No production caller reads it after this change.

**File:** `app/client/core/router/index.ts`

Re-export `queryParams` and `updateQuery` alongside `navigate`:

```ts
export { Router, navigate, queryParams, updateQuery } from "./router";
```

### 2. `hd-document-grid`

**File:** `app/client/ui/modules/document-grid.ts`

Add at module top:

```ts
const DEFAULTS = {
  page: 1,
  pageSize: 12,
  search: "",
  status: "",
  sort: "-UploadedAt",
} as const;
```

Update `@state` initializers to reference `DEFAULTS.*`:

```ts
@state() private page = DEFAULTS.page;
@state() private pageSize = DEFAULTS.pageSize;
@state() private search = DEFAULTS.search;
@state() private status = DEFAULTS.status;
@state() private sort = DEFAULTS.sort;
```

Hydrate in `connectedCallback`:

```ts
connectedCallback() {
  super.connectedCallback();
  this.hydrateFromQuery();
  this.fetchDocuments();
}

private hydrateFromQuery() {
  const q = queryParams();
  if (q.page) this.page = Number(q.page) || DEFAULTS.page;
  if (q.page_size) this.pageSize = Number(q.page_size) || DEFAULTS.pageSize;
  if (q.search) this.search = q.search;
  if (q.status) this.status = q.status;
  if (q.sort) this.sort = q.sort;
}
```

Add a `syncQuery` method, call it from each handler before `fetchDocuments`:

```ts
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

Handler touch points (insert `this.syncQuery()` before the existing fetch call):
- `handleSearchInput` — after the debounce timer fires (inside `setTimeout`), call `this.refresh()` which becomes `page = DEFAULTS.page; syncQuery(); fetchDocuments()`.
- `handleStatusFilter`, `handleSort` — already call `this.refresh()`; folding `syncQuery` into `refresh` keeps it DRY.
- `handlePageChange`, `handlePageSizeChange` — add `this.syncQuery()` before `fetchDocuments()`.

Consolidate via `refresh`:

```ts
async refresh() {
  this.page = DEFAULTS.page;
  this.syncQuery();
  await this.fetchDocuments();
}
```

### 3. `hd-prompt-list`

**File:** `app/client/ui/modules/prompt-list.ts`

Same pattern. Defaults:

```ts
const DEFAULTS = {
  page: 1,
  pageSize: 12,
  search: "",
  stage: "",
  sort: "Name",
} as const;
```

Filter key is `stage` (not `status`), sort default is `"Name"`. Hydration, sync, and handler touch points mirror document-grid.

## Files Modified

- `app/client/core/router/router.ts` — add `queryParams`, `updateQuery`; remove query-attribute splat
- `app/client/core/router/index.ts` — re-export new helpers
- `app/client/ui/modules/document-grid.ts` — `DEFAULTS`, hydrate, sync
- `app/client/ui/modules/prompt-list.ts` — `DEFAULTS`, hydrate, sync

No changes to `api.ts`, `domains/*`, or `pagination-controls.ts`.

## Verification

Manual E2E (UI-first per CLAUDE.md):

1. `bun run watch` + `mise run dev` in two terminals.
2. On `/documents`: change each of search, status, sort, page, page size. URL reflects only non-default values; list re-renders; browser history length stays constant across filter changes (replaceState).
3. Click a document card → `/review/:id` → browser Back. Filter state fully restored.
4. Hard-reload `/documents?page=2&status=review&sort=Filename`. Mounts with those values, single fetch fires.
5. Repeat 2-4 on `/prompts` with `stage` filter, default sort `"Name"`.
6. Default state: no query touched → URL stays `/documents` (no `?`).
7. `bun run build && mise run vet` — no type/lint regressions.

Tests are AI-owned in Phase 5 of the session.
