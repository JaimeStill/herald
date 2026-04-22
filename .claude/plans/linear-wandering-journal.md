# Issue #139 — Bulk delete action for document list

## Context

Post-IL6 quality-of-life work (sub-issue of objective #132). The document list already supports bulk classification via "Classify N Documents"; users should have the same bulk affordance for deletion. Single-card Delete uses `hd-confirm-dialog`; the bulk action must follow suit.

The issue acceptance criteria also require **"per-item failures surface as error toasts and leave failed IDs selected for retry."** No toast/notification infrastructure currently exists in the Herald web client — mutation failures today are either silent (grid/list actions) or shown as inline error divs (forms). The user has directed that this task additionally introduce **a minimal toast service used for all command executions (API calls that mutate state)**, with the bulk-delete failure path specifically keeping `console.error` + retry retention on top of the toast surface.

## Scope (broader than the issue as filed)

1. **Toast infrastructure** — new `hd-toast-container` element + `Toast` service module. Mounted once at the app shell.
2. **Wire every mutation call site** to emit a toast on success and on failure (11 sites across 5 components).
3. **Add bulk-delete action** to `document-grid` using the new toast service + `Promise.allSettled` retry-retention semantics specified by the issue.

## Architecture

### Toast service

File: `app/client/ui/elements/toast.ts` (co-locate element + service; same pattern as existing domain namespaces).

```ts
type ToastKind = "success" | "error" | "warning" | "info";
interface Toast { id: string; kind: ToastKind; message: string; }

// Module-level singleton bus
// Subscribers = ToastContainer instances (usually one)
// push() appends, schedules auto-dismiss via setTimeout
// dismiss() removes by id and notifies subscribers

export const Toast = {
  success(message: string): void,
  error(message: string): void,
  warning(message: string): void,
  info(message: string): void,
};

@customElement("hd-toast-container")
export class ToastContainer extends LitElement { ... }
```

Design points:
- **Auto-dismiss**: success/info = 3s, warning = 5s, error = 6s. Hover pauses dismiss (optional; skip if it bloats scope).
- **Click-to-dismiss**: clicking a toast immediately removes it (cancels the auto-dismiss timer).
- **Stacking**: newest at the bottom of a fixed-position column (`position: fixed; bottom: var(--space-4); right: var(--space-4);`), capped at say 5 visible; overflow removes oldest.
- **Styling**: uses existing semantic tokens — `--green/--green-bg` (success), `--red/--red-bg` (error), `--yellow/--yellow-bg` (warning), `--blue/--blue-bg` (info). Border-left accent + monospace text, matching Herald's terminal aesthetic.
- **Register globally**: `app/client/app.ts` imports the element (side-effect registration). Mount `<hd-toast-container></hd-toast-container>` once in `app/server/layouts/app.html` after `<main>`.

### Bulk delete handler (`document-grid.ts`)

Mirror the bulk-classify template block but use `btn-red` styling and route through a second `hd-confirm-dialog` instance. Introduce a new `@state() private deleteDocuments: Document[] | null` that holds the snapshot of selected documents when the dialog opens (plural mirror of the existing singular `deleteDocument: Document | null`).

```ts
private handleBulkDelete() {
  if (!this.documents) return;
  const selected = this.documents.data.filter((d) => this.selectedIds.has(d.id));
  if (selected.length === 0) return;
  this.deleteDocuments = selected;
}

private async confirmBulkDelete() {
  const batch = this.deleteDocuments;
  this.deleteDocuments = null;
  if (!batch) return;

  const ids = batch.map((d) => d.id);
  const results = await Promise.allSettled(
    ids.map((id) => DocumentService.delete(id)),
  );

  const failed = new Set<string>();
  let succeeded = 0;

  results.forEach((result, i) => {
    const id = ids[i];
    if (result.status === "fulfilled" && result.value.ok) {
      succeeded++;
    } else {
      failed.add(id);
      const error =
        result.status === "rejected"
          ? String(result.reason)
          : result.value.error;
      const doc = batch[i];
      console.error(`Failed to delete document ${id}:`, error);
      Toast.error(`Failed to delete ${doc.filename}: ${error}`);
    }
  });

  if (succeeded > 0) {
    Toast.success(`Deleted ${succeeded} document${succeeded === 1 ? "" : "s"}`);
  }

  this.selectedIds = failed;
  this.fetchDocuments();
}

private cancelBulkDelete() {
  this.deleteDocuments = null;
}
```

Only one `hd-confirm-dialog` is open at a time (bulk vs. single), but render them as two separate conditional blocks for clarity:

```html
${this.deleteDocument
  ? html`<hd-confirm-dialog message="Are you sure you want to delete ${this.deleteDocument.filename}?" ...></hd-confirm-dialog>`
  : nothing}
${this.deleteDocuments
  ? html`<hd-confirm-dialog message="Are you sure you want to delete ${this.deleteDocuments.length} documents?" ...></hd-confirm-dialog>`
  : nothing}
```

Button markup (placed immediately after the bulk-classify conditional block in `renderToolbar`):

```html
${this.selectedIds.size > 0
  ? html`<button class="btn btn-red" @click=${this.handleBulkDelete}>
      Delete ${this.selectedIds.size} Documents
    </button>`
  : nothing}
```

The `btn-red` class already exists in `@styles/buttons.module.css` (line 41) — **no new CSS is needed** for the button itself. The issue's mention of "Delete variant styling for the new button" in `document-grid.module.css` is satisfied by reusing the shared utility; flag this in the session summary.

### Mutation-site toast wiring

Convention applied uniformly:
- **Success**: `Toast.success("<verb past tense> <noun>")` — e.g., `"Document deleted"`, `"Prompt activated"`, `"Classification validated"`.
- **Failure**: `Toast.error(result.error ?? "Unexpected error")`.
- **Forms** (`classification-panel`, `prompt-form`): keep the existing inline `this.error` state (used for form-validation context). Add a toast call alongside, so the global feedback surface is consistent without breaking form UX.
- **SSE classify**: on `onComplete` fire `Toast.success("Classified <filename>")` if we can plumb the filename; otherwise `"Classified document"`. On `onError` fire `Toast.error("Classification failed")`.

## Critical files

| File | Change |
|------|--------|
| `app/client/ui/elements/toast.ts` | **NEW** — `Toast` service + `hd-toast-container` element |
| `app/client/ui/elements/toast.module.css` | **NEW** — toast stack + item styling |
| `app/client/ui/elements/index.ts` | Re-export `ToastContainer` (match existing barrel convention) |
| `app/client/app.ts` | Import `toast.ts` for side-effect registration |
| `app/server/layouts/app.html` | Mount `<hd-toast-container></hd-toast-container>` after `<main>` |
| `app/client/ui/modules/document-grid.ts` | Add bulk-delete button, `deleteDocuments` state, `handleBulkDelete`/`confirmBulkDelete`/`cancelBulkDelete`, second confirm-dialog block, success/error toasts on all existing mutations (delete + classify) |
| `app/client/ui/modules/classification-panel.ts` | Add `Toast.success` / `Toast.error` around `validate()` and `update()` (keep inline error) |
| `app/client/ui/modules/prompt-list.ts` | Add `Toast.success` / `Toast.error` around `activate`/`deactivate`/`delete` |
| `app/client/ui/modules/prompt-form.ts` | Add `Toast.success` / `Toast.error` around `create`/`update` (keep inline error) |
| `app/client/ui/modules/document-upload.ts` | Add `Toast.success`/`Toast.error` summary after batch upload completes |
| `app/client/ui/modules/document-grid.module.css` | **No change** — shared `btn-red` suffices |

## Reused primitives (do not reinvent)

- `DocumentService.delete(id)` — `app/client/domains/documents/service.ts:51` — returns `Result<void>`.
- `hd-confirm-dialog` — `app/client/ui/elements/confirm-dialog.ts` — dispatches `confirm` / `cancel`.
- `@styles/buttons.module.css` — `.btn`, `.btn-red`, `.btn-blue` variants already available.
- Semantic color tokens — `--green`, `--red`, `--yellow`, `--blue` (+ `-bg` variants) in `app/client/design/core/tokens.css:40-57`.
- `Result<T>` shape — `{ ok: true, data } | { ok: false, error: string }` — used by every non-SSE mutation.

## Verification

1. `mise run dev` — start the server with hot reload.
2. **Bulk delete happy path**: upload 3–5 test PDFs from `_project/marked-documents/`, select multiple cards, click "Delete N Documents", confirm. Expect: confirm dialog with correct count, parallel deletes, grid refreshes, success toast `"Deleted N documents"`, selection cleared.
3. **Bulk delete cancel**: open the dialog, click cancel. Expect: dialog closes, selection preserved, no API call, no toast.
4. **Bulk delete partial failure**: simulate failure via DevTools → Network throttling with offline mode toggled mid-operation, or by deleting the same ID twice via two tabs. Expect: `console.error` entries, one error toast per failed ID, success toast for the batch total, failed IDs remain in `selectedIds` and remain checkbox-selected in the grid.
5. **Single-card delete regression**: open a single card's Delete button, confirm. Expect: existing flow intact (dialog with filename, delete, grid refresh), now also emits a success toast.
6. **Bulk classify regression**: select documents, click "Classify N Documents". Expect: existing SSE progress works, on completion each document emits a success toast, on error an error toast fires.
7. **Forms**: save a prompt / validate a classification; confirm inline error state still works AND a toast appears on both success and failure.
8. `mise run vet` — no Go vet regressions (Go surface unchanged, but run it anyway).
9. Web client build (`cd app && bun run build`) — ensure no TS errors.
10. No automated web-client tests exist; validation is manual-plus-typecheck per Herald conventions.

## Risk notes

- **Scope**: this task now touches 5 client modules beyond the originally scoped `document-grid.{ts,module.css}`. Called out here so reviewers know the expanded footprint maps to the user's "toast for all command executions" directive and not scope creep inside the AI.
- **No new CSS in `document-grid.module.css`** despite the issue listing it — the shared `btn-red` covers it. Plan will be explicit about why.
- **Inline form errors kept** to avoid regressing form validation UX. Toast is additive, not replacement.
- **SSE classify toast** requires threading the filename into the callback context; if that's awkward, fall back to a generic message. Low-risk cosmetic.

## CHANGELOG tag at closeout

`v0.5.0-dev.132.139` (objective 132, issue 139).
