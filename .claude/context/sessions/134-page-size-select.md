# 134 - Configurable page size selector for list views

## Summary

Extended `<hd-pagination>` with a per-page size selector (12 / 24 / 48 / 96) and an editable page-number input that replaces the static "Page X of N" indicator. Both `document-grid` and `prompt-list` promoted their hardcoded `page_size: 12` to a `pageSize` `@state()` field and now refetch with `page = 1` when the selector changes.

## Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| New element vs. extend `hd-pagination` | Extend | Pagination and per-page size are one conceptual footer unit — always rendered together, share the same divider. Avoids a new module-level wrapper in both callers and keeps the `border-top` in one place. |
| State field naming | `pageSize` (camelCase) | Matches existing `selectedIds` / `deleteDocument` pattern. `page_size` snake_case stays on the wire in `SearchRequest`. |
| Page-input commit timing | `@change` (blur / Enter) | `@input` would refetch on every keystroke. |
| Invalid input handling | Silent clamp + rebind | `Math.trunc` then clamp to `[1, totalPages]`; non-numeric or empty reverts to `this.page`; DOM `value` is written directly so the input clears the bad text even when `this.page` didn't change. |
| Browser-native behaviors | Keep spinners and wheel-to-change | Match user expectations for `type="number"`. |
| Page input sizing | `width: auto; padding: var(--space-2) var(--space-1);` | Auto width scales with digit count (1 → 4 digits) without truncation risk; reduced horizontal padding prevents a narrow input from consuming excess width. Height stays consistent with other form inputs across the UI (taller than buttons, by choice). |

## Files Modified

- `app/client/ui/elements/pagination-controls.ts` — added `size` / `sizeOptions` properties, `handleSizeChange` / `handlePageInput` / `handlePageFocus`, restructured template into `.page-size` + `.page-controls`, updated class JSDoc
- `app/client/ui/elements/pagination-controls.module.css` — `justify-content: space-between`, new `.page-size` / `.page-controls` / `.label` / `.page-input` rules, `.page-indicator` promoted to flex row
- `app/client/ui/modules/document-grid.ts` — added `pageSize` state, wired `.size` + `@page-size-change` on `<hd-pagination>`, added `handlePageSizeChange` (resets page to 1 + refetches)
- `app/client/ui/modules/prompt-list.ts` — same wiring as document-grid

## Patterns Established

- **Editable numeric fields** — pattern for `<input type="number">` bound to Lit state: bind `.value=${String(this.x)}`, commit on `@change` with `valueAsNumber` guard, trunc-then-clamp, silent rebind on invalid/no-op (write `target.value` directly to clear stale DOM text), disable at degenerate bounds (e.g., `totalPages <= 1`), `aria-label` since surrounding text isn't programmatically associated.
- **Multi-concern footer elements** — when two pieces of UI always render together as a conceptual unit (pagination + page size), extend a single element rather than introducing a sibling. Keeps caller templates one element wide and divider/spacing concerns in one place.

## Validation Results

- `mise run test` — all 21 Go test packages pass.
- `mise run vet` — clean.
- `mise run web:build` — `dist/app.js` + `dist/app.css` built without TS errors.
- Manual verification on `/app/documents` (3-page, 24-entry list): per-page selector cycles [12, 24, 48, 96], list refetches and resets to page 1 each time; page input accepts direct page numbers, clamps out-of-range values, reverts on non-numeric; spinner arrows and mouse-wheel stepping work within `min`/`max`; auto-select on focus behaves correctly. Screenshot captured showing the "Per page: 12 / Prev / Page [1] / of 3 / Next" footer layout with the divider spanning the full row.
