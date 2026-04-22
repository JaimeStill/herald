# 135 - URL query-parameter state persistence for list views

## Summary

Two-way URL binding for `hd-document-grid` and `hd-prompt-list` filter state. Views hydrate pagination, search, filter, and sort from the URL query string on mount via a new `queryParams()` router helper, and write them back on every change via `updateQuery()` — which uses `history.replaceState` to avoid remounting the active view. Defaults are omitted from the URL so shareable links stay clean. Navigating into `/review/:id` and returning, or reloading the page, now restores the prior grid state exactly.

Paired with a visual simplification of `hd-pagination` (folded in as remediation R1 after the developer observed that the full-width layout did not scale into the narrow prompt-list column): chevron buttons replace textual prev/next, "Page X of N" collapses to `[input] / N`, native number-input spinners are hidden so the page input auto-sizes to its digit count, and prev/next `align-self: stretch` to match the sibling input/select height. The component now renders the same in both the wide documents toolbar and the narrow prompt-list column.

## Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| State decorator for filter fields | Keep as `@state`, not `@property` | `@property` is reserved for parent-passed inputs across the codebase. Filter state is view-internal — converting would blur that boundary. |
| URL ingress path | Explicit `queryParams()` helper in `connectedCallback` | The router's attribute-splat for query keys was unused by any view. Explicit hydration makes the contract visible and keeps `@state` fields private. |
| URL egress path | `updateQuery()` built on `URLSearchParams` + `history.replaceState` | `pushState` would stack history entries per keystroke; the router's `innerHTML = ""` teardown would remount the view on any route mutation. `replaceState` avoids both. |
| Reuse of `toQueryString` | No, build `updateQuery` independently | `toQueryString` is shaped for API request URLs (prepends `?`, flat params object). URL state sync needs merge-and-delete semantics over an existing URL. |
| Default-value location | Per-view `DEFAULTS` const at module top | Single source of truth for field initializers, hydration fallbacks, and sync-omission checks. App-wide defaults would require server-injected config — out of scope. |
| Pagination controls layout | Simplify permanently, not dual-mode | User preference after weighing container-query complexity vs. a simpler universal layout. Nothing is lost functionally; a11y preserved via `aria-label` on the chevron buttons. |
| Page input width | Keep `width: auto` | Intrinsic sizing scales with digit count — important as the corpus grows toward 750k-1M documents. Native spinner buttons hidden via `appearance: textfield` so the reserved space collapses. |

## Files Modified

- `app/client/core/router/router.ts` — add `queryParams` and `updateQuery` exports, remove query-attribute splat in `mount()`, refresh `Router` class docstring
- `app/client/core/router/index.ts` — re-export the two new helpers
- `app/client/ui/modules/document-grid.ts` — `DEFAULTS` const, `hydrateFromQuery`/`syncQuery` methods, handler sync points
- `app/client/ui/modules/prompt-list.ts` — same pattern (filter is `stage`, sort default is `"Name"`)
- `app/client/ui/elements/pagination-controls.ts` — chevron glyphs + aria-labels, restored "Page Size" label, `[input] / N` indicator
- `app/client/ui/elements/pagination-controls.module.css` — `.page-size-label`, `.btn-page` with `align-self: stretch` + `--text-xl` glyphs, spinner-removal rules on `.page-input`
- `.claude/skills/web-development/SKILL.md` — update router one-liner to reflect the new parameter-passing contract
- `.claude/skills/web-development/references/router.md` — rewrite "Parameter Passing" section to document the path-attribute / query-helper split
- `CHANGELOG.md` — new `v0.5.0-dev.132.135` section

## Patterns Established

- **Query-state pattern for list views**: `DEFAULTS` const at module top → `hydrateFromQuery()` in `connectedCallback` → `syncQuery()` called before every `fetch*()` in handlers → `refresh()` folds page reset + sync + fetch. Reuse this shape for any future list view with URL-persistent state.
- **Router helper split**: `navigate` (route change with remount), `queryParams` (read current URL state), `updateQuery` (write URL state without remount). These three cover the full navigation-and-state surface for the app.
- **Height matching via `align-self: stretch`**: cleaner than hardcoded heights or magic padding numbers — the flex line's cross-axis size is already determined by the tallest sibling, so opting into stretch gives automatic alignment.

## Validation Results

- `bun run build` — clean, `dist/app.js` and `dist/app.css` emitted.
- `mise run vet` — clean (`go vet ./...` reports nothing).
- Manual UI verification by the developer (screenshots reviewed mid-session):
  - Documents view: search / status / sort / page / page_size each write to URL with non-default values only; defaults omitted; browser history length constant across filter changes.
  - Navigate `/documents` → `/review/:id` → Back: grid state restored.
  - Hard-reload with query params (`?page=2&status=review`): mounts with those values.
  - Prompts view: same behavior with `stage` filter and `"Name"` sort default.
  - Pagination controls render cleanly in both the wide documents toolbar and narrow prompt-list column.
- **No web client test authorship**: the project has no TypeScript/Lit test harness (no bun test files, no jsdom, no vitest). Adding a framework is out of scope for this task. Per CLAUDE.md, Go tests in `tests/` mirror `internal/`; web client coverage has historically relied on manual visual QA. The changes here — browser-API-heavy router helpers and Lit lifecycle wiring — are exactly the surfaces that a browser-harness framework would be needed to cover. Flagging as a follow-up consideration.
