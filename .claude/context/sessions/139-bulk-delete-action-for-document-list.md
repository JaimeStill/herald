# 139 - Bulk delete action for document list

## Summary

Added a "Delete N Documents" bulk action to the document grid, mirroring the existing "Classify N Documents" bulk action. Per the issue, per-item failures surface as error toasts and leave failed IDs selected for retry. Scope expanded at the user's direction to introduce a minimal toast service used for all command executions (API calls that mutate state); every existing mutation call site in the web client now emits success and failure toasts.

A follow-up task was filed during this session (#145) to adopt native overlay primitives — `<dialog>.showModal()` for `hd-confirm-dialog`, `popover="manual"` for `hd-toast-container`, and a new `<hd-tooltip>` using `popover="hint"` — and the overlay convention is now documented in CLAUDE.md and the web-development skill.

## Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Toast service surface | Global namespace (`Toast.success`, `Toast.error`, etc.) + `<hd-toast-container>` element | Matches existing domain-service pattern (`DocumentService.*`); element subscribes to a module-level bus so any call site can fire without importing the element |
| Error handling for bulk delete | `Promise.all` over per-doc promises that pair `{ doc, ok, error? }` | Eliminates parallel-array index bookkeeping between ids/batch/results; a future edit that reorders one array can't silently corrupt the others |
| Retry semantics | Failed IDs remain in `selectedIds`; successful IDs drop out | Directly matches issue acceptance criterion "leave failed IDs selected for retry" |
| Dialog rendering | Two separate conditional `<hd-confirm-dialog>` blocks (one per state) | Clearer than overloading a single dialog with branching logic; only one renders at a time |
| Button placement | "Delete N Documents" adjacent to "Classify N Documents" in the same conditional block | Both bulk actions share identical visibility rule (`selectedIds.size > 0`) |
| No new CSS in `document-grid.module.css` | Reuse shared `.btn-red` from `@styles/buttons.module.css` | The issue mentioned a "Delete variant" but the existing utility already matches the card Delete styling |
| Toast container mount point | Programmatic `document.body.appendChild` from `app.ts` after Router starts | Matches Herald's pattern (user menu is also wired dynamically); keeps `app/server/layouts/app.html` a pure shell |
| Toast stack width | Fixed `min(72ch, calc(100dvw - var(--space-8)))` | Consistent toast width regardless of content; `ch` hugs the monospace font, `dvw` tracks the dynamic viewport |
| Click-to-dismiss | Clicking any toast cancels its timer and removes it | Lightweight user control without inventing a close button |
| Forms keep inline error | `classification-panel` and `prompt-form` retain their `this.error` state alongside toasts | Inline errors anchor to form context; toasts are the global feedback layer — additive, not replacement |

## Files Modified

**New:**
- `app/client/ui/elements/toast.ts` — `Toast` service + `<hd-toast-container>` element
- `app/client/ui/elements/toast.module.css` — bottom-center fixed stack + kind-specific styling

**Modified:**
- `app/client/ui/elements/index.ts` — re-export `Toast`, `ToastContainer`, `ToastItem`, `ToastKind`
- `app/client/app.ts` — mount `<hd-toast-container>` via `document.body.appendChild`
- `app/client/ui/modules/document-grid.ts` — `deleteDocuments` state, `handleBulkDelete`/`confirmBulkDelete`/`cancelBulkDelete`, second confirm-dialog block, bulk-delete button, toast wiring on single-delete and SSE classify callbacks
- `app/client/ui/modules/classification-panel.ts` — toast wiring around `validate()` and `update()` (inline error retained)
- `app/client/ui/modules/prompt-list.ts` — toast wiring around `activate`/`deactivate`/`delete`
- `app/client/ui/modules/prompt-form.ts` — toast wiring around `create`/`update` (inline error retained)
- `app/client/ui/modules/document-upload.ts` — batch toast summary after `Promise.allSettled`
- `.claude/CLAUDE.md` — overlay-primitives convention under Web Client Conventions
- `.claude/skills/web-development/SKILL.md` — new "Overlay Convention" section with decision matrix and patterns

## Patterns Established

- **Command-execution feedback**: Every mutation API call surfaces success + failure via `Toast.success` / `Toast.error`. Error messages follow the shape `Failed to <verb> <noun>: <result.error>`; success uses `<past-tense-verb> <noun>`.
- **Toast service**: Module-level namespace + subscribable bus, mirroring the stateless domain-service pattern. Element mount happens in `app.ts`, not `app.html`.
- **Promise-all with carried payload**: For bulk operations that need both `Promise.allSettled` safety and per-item context, map each input to `async (item) => ({ item, ok, error? })` inside a `try`/`catch` and iterate the outcome array once. No parallel-array zipping.
- **Overlay primitives convention**: Any UI that overlays the page (tooltips, menus, toasts, modals) must use `<dialog>` or the Popover API — no manual `position: fixed; z-index` overlay divs. Captured in CLAUDE.md and SKILL.md; implemented across the three relevant elements in the follow-up task #145.

## Validation Results

- `(cd app && bun run build)` — passes, no TS errors; artifacts: `dist/app.js`, `dist/app.css`.
- `mise run vet` — passes (Go surface unchanged, verified as a sanity check).
- Manual verification in the browser (screenshots provided by user):
  - Bulk delete button appears adjacent to bulk classify when ≥1 document selected.
  - Confirm dialog renders with "Are you sure you want to delete {N} documents?".
  - Parallel deletes complete; grid refreshes; one success toast per batch.
  - Prompt activate/deactivate toasts observed live.
- No automated web-client tests exist in Herald; validation is build + manual per project convention.

## Known Gaps / Follow-ups

- **hd-toast-container `z-index: 200`** will collide with modals once #145 lands (which moves `hd-confirm-dialog` to top-layer via `<dialog>.showModal()`). Resolved there by refactoring the stack to `popover="manual"`.
- **Single-delete error path initially had a missing space** (`Failed to delete${filename}`) and **classification-panel update failure was missing a toast call**; both caught during closeout validation and fixed before commit.
