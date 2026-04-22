# 139 - Bulk delete action for document list

## Problem Context

Herald's document list already supports bulk classification via "Classify N Documents". Users need the same bulk affordance for deletion. Single-card Delete uses `hd-confirm-dialog`, and the bulk action must follow suit — confirming with "Are you sure you want to delete {N} documents?", deleting in parallel via `Promise.allSettled`, and retaining failed IDs in the selection for retry.

Issue #139 acceptance criteria also specify that per-item failures surface as **error toasts**. Herald currently has no toast/notification infrastructure — grid/list mutations fail silently, and forms use inline error divs. This session therefore introduces a minimal toast service used for **all** command executions (API calls that mutate state) and wires every existing mutation call site through it, alongside the bulk-delete work.

## Architecture Approach

### Toast service

A new `app/client/ui/elements/toast.ts` hosts both the element (`<hd-toast-container>`) and the service (`Toast` namespace). A module-level `ToastBus` singleton holds the toast array, notifies subscribed elements, and manages per-toast auto-dismiss timers.

- **Kinds**: `success`, `error`, `warning`, `info`. Auto-dismiss durations: 3s success/info, 5s warning, 6s error.
- **Click-to-dismiss**: clicking a toast cancels its timer and removes it.
- **Stacking**: fixed bottom-center, newest at the bottom, max 5 visible (oldest evicted).
- **Styling**: semantic border-left accent + matching `-bg` background, monospace text, consistent with Herald's terminal aesthetic.
- **Mount**: `<hd-toast-container>` is appended once to `document.body` from `app/client/app.ts` after the Router starts — matching Herald's existing pattern (the user menu is also wired dynamically from `app.ts`, and `app.html` stays a pure server-rendered shell). Element registration still flows through the `@ui/elements` barrel.

### Bulk-delete state model

`document-grid.ts` gains a second delete state `deleteDocuments: Document[] | null` that mirrors the existing singular `deleteDocument: Document | null`. Opening the bulk dialog snapshots the currently-selected documents (so the count shown in the dialog does not shift if selection changes while open). Only one dialog renders at a time.

On confirm: `Promise.allSettled` over `DocumentService.delete(id)`, then:
- Successful IDs drop out of `selectedIds`.
- Failed IDs are retained in `selectedIds` for retry; each failure fires `Toast.error` and logs via `console.error`.
- A single `Toast.success("Deleted N documents")` fires for the succeeded batch.
- `fetchDocuments()` refreshes the grid.

### Cross-cutting toast wiring

Convention applied to every mutation:
- **Success**: `Toast.success("<verb past-tense> <noun>")`.
- **Failure**: `Toast.error("<context>: <result.error>")`.
- **Forms** (`classification-panel`, `prompt-form`): keep the existing inline `this.error` display for form-local context; toasts are additive.
- **SSE classify** in `document-grid`: look the filename up from the current page (`this.documents?.data`) and fire `Toast.success("Classified <filename>")` / `Toast.error("Classification failed for <filename>")` inside the existing `onComplete` / `onError` callbacks.
- **Batch upload** in `document-upload`: after `Promise.allSettled` resolves, summarize with one success toast for the succeeded count and one error toast for the failed count.

### No new CSS in `document-grid.module.css`

The issue mentions "Delete variant styling for the new button" in `document-grid.module.css`, but the shared `.btn-red` utility (in `@styles/buttons.module.css`) already matches the `document-card` Delete styling. No scoped CSS is required; call this out in the session summary.

## Implementation

### Step 1: Create `app/client/ui/elements/toast.module.css`

```css
.stack {
  position: fixed;
  bottom: var(--space-4);
  left: 0;
  right: 0;
  margin-inline: auto;
  width: min(72ch, calc(100dvw - var(--space-8)));
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  z-index: 200;
  pointer-events: none;
}

.toast {
  pointer-events: auto;
  display: block;
  width: 100%;
  padding: var(--space-3) var(--space-4);
  border: 1px solid var(--divider);
  border-left-width: 4px;
  border-radius: var(--radius-sm);
  background: var(--bg-1);
  color: var(--color);
  font-family: var(--font-mono);
  font-size: var(--text-xs);
  text-align: left;
  white-space: normal;
  overflow-wrap: break-word;
  cursor: pointer;
  box-shadow: var(--shadow-md);
  animation: slide-in 0.15s ease-out;
  transition: opacity 0.15s;
}

.toast:hover {
  opacity: 0.9;
}

.toast.success {
  border-left-color: var(--green);
  background: var(--green-bg);
}

.toast.error {
  border-left-color: var(--red);
  background: var(--red-bg);
}

.toast.warning {
  border-left-color: var(--yellow);
  background: var(--yellow-bg);
}

.toast.info {
  border-left-color: var(--blue);
  background: var(--blue-bg);
}

@keyframes slide-in {
  from {
    opacity: 0;
    transform: translateX(20%);
  }
  to {
    opacity: 1;
    transform: translateX(0);
  }
}
```

### Step 2: Create `app/client/ui/elements/toast.ts`

```ts
import { LitElement, html, nothing } from "lit";
import { customElement, state } from "lit/decorators.js";

import styles from "./toast.module.css";

export type ToastKind = "success" | "error" | "warning" | "info";

export interface ToastItem {
  id: string;
  kind: ToastKind;
  message: string;
}

type Listener = (toasts: ToastItem[]) => void;

const DURATIONS: Record<ToastKind, number> = {
  success: 3000,
  info: 3000,
  warning: 5000,
  error: 6000,
};

const MAX_VISIBLE = 5;

class ToastBus {
  private toasts: ToastItem[] = [];
  private listeners = new Set<Listener>();
  private timers = new Map<string, number>();

  subscribe(fn: Listener): () => void {
    this.listeners.add(fn);
    fn(this.toasts);
    return () => {
      this.listeners.delete(fn);
    };
  }

  push(kind: ToastKind, message: string): string {
    const id = crypto.randomUUID();
    this.toasts = [...this.toasts, { id, kind, message }];

    while (this.toasts.length > MAX_VISIBLE) {
      const oldest = this.toasts[0];
      this.clearTimer(oldest.id);
      this.toasts = this.toasts.slice(1);
    }

    this.timers.set(
      id,
      window.setTimeout(() => this.dismiss(id), DURATIONS[kind]),
    );
    this.emit();
    return id;
  }

  dismiss(id: string): void {
    this.clearTimer(id);
    const next = this.toasts.filter((t) => t.id !== id);
    if (next.length === this.toasts.length) return;
    this.toasts = next;
    this.emit();
  }

  private clearTimer(id: string) {
    const timer = this.timers.get(id);
    if (timer !== undefined) {
      clearTimeout(timer);
      this.timers.delete(id);
    }
  }

  private emit() {
    for (const fn of this.listeners) fn(this.toasts);
  }
}

const bus = new ToastBus();

export const Toast = {
  success(message: string): string {
    return bus.push("success", message);
  },
  error(message: string): string {
    return bus.push("error", message);
  },
  warning(message: string): string {
    return bus.push("warning", message);
  },
  info(message: string): string {
    return bus.push("info", message);
  },
  dismiss(id: string): void {
    bus.dismiss(id);
  },
  subscribe(fn: Listener): () => void {
    return bus.subscribe(fn);
  },
};

@customElement("hd-toast-container")
export class ToastContainer extends LitElement {
  static styles = [styles];

  @state() private toasts: ToastItem[] = [];

  private unsubscribe?: () => void;

  connectedCallback() {
    super.connectedCallback();
    this.unsubscribe = Toast.subscribe((toasts) => {
      this.toasts = toasts;
    });
  }

  disconnectedCallback() {
    super.disconnectedCallback();
    this.unsubscribe?.();
    this.unsubscribe = undefined;
  }

  private handleDismiss(id: string) {
    Toast.dismiss(id);
  }

  render() {
    if (this.toasts.length === 0) return nothing;

    return html`
      <div class="stack" role="status" aria-live="polite">
        ${this.toasts.map(
          (t) => html`
            <button
              type="button"
              class="toast ${t.kind}"
              @click=${() => this.handleDismiss(t.id)}
            >
              ${t.message}
            </button>
          `,
        )}
      </div>
    `;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    "hd-toast-container": ToastContainer;
  }
}
```

### Step 3: Export `ToastContainer` from the elements barrel

Edit `app/client/ui/elements/index.ts` — add the re-export alphabetically (after `PromptCard`):

```ts
export { PromptCard } from "./prompt-card";
export { Toast, ToastContainer } from "./toast";
export type { ToastItem, ToastKind } from "./toast";
```

### Step 4: Mount the toast container from `app.ts`

Edit `app/client/app.ts` — after `router.start()` (line 20), append a `<hd-toast-container>` to the document body. Matches the dynamic-mount pattern already used for the user menu below it.

```ts
  const router = new Router("app-content", routes);
  router.start();

  document.body.appendChild(document.createElement("hd-toast-container"));

  if (Auth.isEnabled()) {
```

`app/server/layouts/app.html` is **not** modified — the server template stays a pure shell.

### Step 5: Bulk delete + toast wiring in `document-grid.ts`

1. Add a `Toast` import alongside the other `@ui/elements` usage:

```ts
import { Toast } from "@ui/elements";
```

2. Add a new state field next to the existing `deleteDocument`:

```ts
@state() private deleteDocument: Document | null = null;
@state() private deleteDocuments: Document[] | null = null;
```

3. Update the existing `confirmDelete` to emit toasts (replace the current method body):

```ts
private async confirmDelete() {
  if (!this.deleteDocument) return;

  const id = this.deleteDocument.id;
  const filename = this.deleteDocument.filename;
  this.deleteDocument = null;

  const result = await DocumentService.delete(id);

  if (result.ok) {
    this.selectedIds.delete(id);
    this.fetchDocuments();
    Toast.success(`Deleted ${filename}`);
  } else {
    Toast.error(`Failed to delete ${filename}: ${result.error}`);
  }
}
```

4. Add the three new handlers immediately after `handleBulkClassify`:

```ts
private handleBulkDelete() {
  if (!this.documents) return;

  const selected = this.documents.data.filter((d) =>
    this.selectedIds.has(d.id),
  );
  if (selected.length === 0) return;

  this.deleteDocuments = selected;
}

private async confirmBulkDelete() {
  const batch = this.deleteDocuments;
  this.deleteDocuments = null;
  if (!batch) return;

  const outcomes = await Promise.all(
    batch.map(async (doc) => {
      try {
        const result = await DocumentService.delete(doc.id);
        return result.ok
          ? { doc, ok: true as const }
          : { doc, ok: false as const, error: result.error };
      } catch (err) {
        return { doc, ok: false as const, error: String(err) };
      }
    }),
  );

  const failed = new Set<string>();
  let succeeded = 0;

  for (const outcome of outcomes) {
    if (outcome.ok) {
      succeeded++;
      continue;
    }
    failed.add(outcome.doc.id);
    console.error(
      `Failed to delete document ${outcome.doc.id}:`,
      outcome.error,
    );
    Toast.error(`Failed to delete ${outcome.doc.filename}: ${outcome.error}`);
  }

  if (succeeded > 0) {
    Toast.success(
      `Deleted ${succeeded} document${succeeded === 1 ? "" : "s"}`,
    );
  }

  this.selectedIds = failed;
  this.fetchDocuments();
}

private cancelBulkDelete() {
  this.deleteDocuments = null;
}
```

5. Extend the SSE classify callbacks in `handleClassify` to emit toasts. Replace the existing `onComplete` and `onError` bodies with:

```ts
onComplete: () => {
  this.abortControllers.delete(docId);
  const updated = new Map(this.classifying);
  updated.delete(docId);
  this.classifying = updated;
  const filename =
    this.documents?.data.find((d) => d.id === docId)?.filename ??
    "document";
  Toast.success(`Classified ${filename}`);
  this.fetchDocuments();
},
onError: () => {
  this.abortControllers.delete(docId);
  const updated = new Map(this.classifying);
  updated.delete(docId);
  this.classifying = updated;
  const filename =
    this.documents?.data.find((d) => d.id === docId)?.filename ??
    "document";
  Toast.error(`Classification failed for ${filename}`);
  this.fetchDocuments();
},
```

6. Add the bulk-delete button to `renderToolbar`. Insert immediately after the existing bulk-classify block so both live in the same conditional region:

```ts
${this.selectedIds.size > 0
  ? html`
      <button class="btn btn-blue" @click=${this.handleBulkClassify}>
        Classify ${this.selectedIds.size} Documents
      </button>
      <button class="btn btn-red" @click=${this.handleBulkDelete}>
        Delete ${this.selectedIds.size} Documents
      </button>
    `
  : nothing}
```

7. Add the second confirm-dialog block. In the `render()` method, right after the existing `${this.deleteDocument ? html\`<hd-confirm-dialog .../>\` : nothing}` block, append:

```ts
${this.deleteDocuments
  ? html`
      <hd-confirm-dialog
        message="Are you sure you want to delete ${this.deleteDocuments
          .length} documents?"
        @confirm=${this.confirmBulkDelete}
        @cancel=${this.cancelBulkDelete}
      ></hd-confirm-dialog>
    `
  : nothing}
```

### Step 6: Toast wiring in `classification-panel.ts`

1. Add `Toast` import:

```ts
import { Toast } from "@ui/elements";
```

2. Update `handleValidate` failure + success paths (keep the inline `this.error` behavior):

```ts
if (!result.ok) {
  this.error = result.error;
  Toast.error(`Validation failed: ${result.error}`);
  return;
}

this.classification = result.data;
this.mode = "view";
this.error = "";

Toast.success("Classification validated");

this.dispatchEvent(
  new CustomEvent("validate", {
    detail: { classification: result.data },
    bubbles: true,
    composed: true,
  }),
);
```

3. Update `handleUpdate` identically:

```ts
if (!result.ok) {
  this.error = result.error;
  Toast.error(`Update failed: ${result.error}`);
  return;
}

this.classification = result.data;
this.mode = "view";
this.error = "";

Toast.success("Classification updated");

this.dispatchEvent(
  new CustomEvent("update", {
    detail: { classification: result.data },
    bubbles: true,
    composed: true,
  }),
);
```

### Step 7: Toast wiring in `prompt-list.ts`

1. Add `Toast` import:

```ts
import { Toast } from "@ui/elements";
```

2. Replace `handleToggleActive` body:

```ts
private async handleToggleActive(
  e: CustomEvent<{ id: string; active: boolean }>,
) {
  const { id, active } = e.detail;
  const result = active
    ? await PromptService.activate(id)
    : await PromptService.deactivate(id);

  if (result.ok) {
    Toast.success(`Prompt ${active ? "activated" : "deactivated"}`);
    this.fetchPrompts();
  } else {
    Toast.error(
      `Failed to ${active ? "activate" : "deactivate"} prompt: ${result.error}`,
    );
  }
}
```

3. Replace `confirmDelete` body:

```ts
private async confirmDelete() {
  if (!this.deletePrompt) return;

  const id = this.deletePrompt.id;
  const name = this.deletePrompt.name;
  this.deletePrompt = null;

  const result = await PromptService.delete(id);

  if (result.ok) {
    Toast.success(`Deleted ${name}`);
    this.dispatchEvent(
      new CustomEvent("delete", {
        detail: { id },
        bubbles: true,
        composed: true,
      }),
    );
    this.fetchPrompts();
  } else {
    Toast.error(`Failed to delete ${name}: ${result.error}`);
  }
}
```

### Step 8: Toast wiring in `prompt-form.ts`

1. Add `Toast` import:

```ts
import { Toast } from "@ui/elements";
```

2. Update `handleSubmit` failure + success paths (keep the inline error):

```ts
this.submitting = false;

if (!result.ok) {
  this.error = result.error ?? "An unexpected error occurred.";
  Toast.error(
    `Failed to ${this.isEdit ? "update" : "create"} prompt: ${this.error}`,
  );
  return;
}

Toast.success(`Prompt ${this.isEdit ? "updated" : "created"}`);

this.dispatchEvent(
  new CustomEvent("save", {
    detail: { prompt: result.data },
    bubbles: true,
    composed: true,
  }),
);
```

### Step 9: Toast summary in `document-upload.ts`

1. Add `Toast` import:

```ts
import { Toast } from "@ui/elements";
```

2. Replace the tail of `handleUpload` (from `this.uploading = false;` onward):

```ts
this.uploading = false;

const succeeded = results.filter((r) => r.status === "fulfilled").length;
const failed = results.length - succeeded;

if (succeeded > 0) {
  Toast.success(`Uploaded ${succeeded} file${succeeded === 1 ? "" : "s"}`);
  this.dispatchEvent(
    new CustomEvent("upload-complete", {
      bubbles: true,
      composed: true,
    }),
  );
}

if (failed > 0) {
  Toast.error(`Failed to upload ${failed} file${failed === 1 ? "" : "s"}`);
}
```

Note: the existing code dispatched `upload-complete` when at least one file succeeded; preserve that semantic by guarding the dispatch inside `succeeded > 0`.

## Validation Criteria

- [ ] `<hd-toast-container>` renders once at the shell and disappears when no toasts are active.
- [ ] Clicking any toast dismisses it immediately (timer cancelled, no double-dismiss error).
- [ ] More than 5 simultaneous toasts: oldest are evicted so the stack stays at 5.
- [ ] Bulk-delete button renders only when `selectedIds.size > 0` and sits adjacent to the bulk-classify button with red (`btn-red`) styling.
- [ ] Clicking "Delete N Documents" opens `hd-confirm-dialog` with message "Are you sure you want to delete {N} documents?".
- [ ] Confirm deletes all selected documents in parallel (`Promise.allSettled`), clears the succeeded IDs from selection, and fires one success toast.
- [ ] On partial failure, each failed ID remains in `selectedIds` (its card stays checkbox-selected after grid refresh), one error toast per failed ID, and `console.error` logged per failure.
- [ ] Cancel dismisses the bulk dialog with no side effects (selection preserved, no API call, no toast).
- [ ] Single-card delete still works: confirm dialog with filename, refresh, now also emits a success toast.
- [ ] Bulk classify still works and emits a success toast on each completion, error toast on each failure.
- [ ] Prompt activate/deactivate/delete/create/update all surface success + error toasts.
- [ ] Classification validate/update surface success + error toasts while preserving the existing inline error div.
- [ ] Batch upload emits one success toast and/or one error toast summarizing the batch.
- [ ] `bun run build` (inside `app/`) completes without TypeScript errors.
- [ ] `mise run vet` passes (Go surface unchanged but verify).
