# Component Patterns

Herald uses a three-tier component hierarchy. Each tier has a distinct responsibility, and crossing boundaries (e.g., a pure element calling an API) creates hidden coupling that makes components harder to test and reuse.

## View Component (composes modules, manages view-level state)

Views are route-level components. They compose modules, manage view-level UI state with `@state()`, and coordinate between modules via `querySelector` and events.

```typescript
import { LitElement, html, nothing } from "lit";
import { customElement, state } from "lit/decorators.js";

import buttonStyles from "@styles/buttons.module.css";
import styles from "./documents-view.module.css";

/** Route-level view that composes the document upload and grid modules. */
@customElement("hd-documents-view")
export class DocumentsView extends LitElement {
  static styles = [buttonStyles, styles];

  @state() private showUpload = false;

  private handleUploadComplete() {
    this.showUpload = false;
    this.renderRoot.querySelector("hd-document-grid")?.refresh();
  }

  render() {
    return html`
      <div class="view">
        <div class="view-header">
          <h1>Documents</h1>
          <button
            class="btn upload-toggle"
            @click=${() => (this.showUpload = !this.showUpload)}
          >
            ${this.showUpload ? "Close" : "Upload"}
          </button>
        </div>
        ${this.showUpload
          ? html`
              <hd-document-upload
                @upload-complete=${this.handleUploadComplete}
              ></hd-document-upload>
            `
          : nothing}
        <hd-document-grid></hd-document-grid>
      </div>
    `;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    "hd-documents-view": DocumentsView;
  }
}
```

**View responsibilities:**
- Manage view-level toggles (e.g., `showUpload`)
- Compose modules as child elements
- Coordinate between modules: relay events, call `querySelector` + public methods
- Keep logic minimal — delegate data concerns to modules

## Stateful Component (module — owns state, calls services)

Modules are self-contained capability units. They own their data via `@state()`, call services directly, manage search/filter/pagination state, and orchestrate child elements. Modules are the workhorses of the UI.

```typescript
import { LitElement, html, nothing } from "lit";
import { customElement, state } from "lit/decorators.js";

import type { PageResult } from "@core";
import { navigate } from "@core/router";
import { ClassificationService } from "@domains/classifications";
import type { WorkflowStage } from "@domains/classifications";
import { DocumentService } from "@domains/documents";
import type { Document, SearchRequest } from "@domains/documents";

import buttonStyles from "@styles/buttons.module.css";
import styles from "./document-grid.module.css";

interface ClassifyProgress {
  currentNode: WorkflowStage | null;
  completedNodes: WorkflowStage[];
}

@customElement("hd-document-grid")
export class DocumentGrid extends LitElement {
  static styles = [buttonStyles, styles];

  @state() private documents: PageResult<Document> | null = null;
  @state() private page = 1;
  @state() private search = "";
  @state() private status = "";
  @state() private sort = "-UploadedAt";
  @state() private classifying = new Map<string, ClassifyProgress>();
  @state() private selectedIds = new Set<string>();
  @state() private deleteDocument: Document | null = null;

  private searchTimer = 0;
  private abortControllers = new Map<string, AbortController>();

  connectedCallback() {
    super.connectedCallback();
    this.fetchDocuments();
  }

  disconnectedCallback() {
    super.disconnectedCallback();
    clearTimeout(this.searchTimer);
    for (const controller of this.abortControllers.values()) {
      controller.abort();
    }
  }

  async refresh() {
    this.page = 1;
    await this.fetchDocuments();
  }

  private async fetchDocuments() {
    const req: SearchRequest = {
      page: this.page,
      page_size: 12,
      sort: this.sort,
    };

    if (this.search) req.search = this.search;
    if (this.status) req.status = this.status;

    const result = await DocumentService.search(req);
    if (result.ok) this.documents = result.data;
  }

  private handleSearchInput(e: Event) {
    const input = e.target as HTMLInputElement;
    this.search = input.value;

    clearTimeout(this.searchTimer);
    this.searchTimer = window.setTimeout(() => {
      this.page = 1;
      this.fetchDocuments();
    }, 300);
  }

  // ... filter, sort, pagination, classify SSE orchestration, delete handlers
}
```

**Module responsibilities:**
- Own all data state via `@state()` — fetched results, filters, pagination, progress maps
- Call services directly in event handlers and lifecycle methods
- Expose a public `refresh()` method for parent views to trigger re-fetch
- Manage SSE streaming lifecycle for their subtree (see Streaming section below)
- Pass data to pure elements via `@property()` bindings
- Listen for custom events from child elements

## Pure Element (stateless)

Pure elements receive data via properties and communicate upward through events. They can import immutable domain infrastructure (types, constants, formatters) but never anything that holds or mutates state (services, context, router).

```typescript
import { LitElement, html, nothing } from "lit";
import { customElement, property } from "lit/decorators.js";

import { formatBytes, formatDate } from "@core/formatting";
import type { WorkflowStage } from "@domains/classifications";
import type { Document } from "@domains/documents";

import badgeStyles from "@styles/badge.module.css";
import buttonStyles from "@styles/buttons.module.css";
import styles from "./document-card.module.css";

@customElement("hd-document-card")
export class DocumentCard extends LitElement {
  static styles = [buttonStyles, badgeStyles, styles];

  @property({ type: Object }) document!: Document;
  @property({ type: Boolean }) classifying = false;
  @property() currentNode: WorkflowStage | null = null;
  @property({ type: Array }) completedNodes: WorkflowStage[] = [];
  @property({ type: Boolean }) selected = false;

  private handleClassify() {
    this.dispatchEvent(
      new CustomEvent("classify", {
        detail: { id: this.document.id },
        bubbles: true,
        composed: true,
      }),
    );
  }

  private handleReview() {
    this.dispatchEvent(
      new CustomEvent("review", {
        detail: { id: this.document.id },
        bubbles: true,
        composed: true,
      }),
    );
  }

  render() {
    const doc = this.document;
    return html`
      <div class="card ${this.selected ? "selected" : ""}">
        <div class="header">
          <span class="filename">${doc.filename}</span>
          <span class="badge ${doc.status}">${doc.status}</span>
        </div>
        ${this.classifying
          ? html`<hd-classify-progress
              .currentNode=${this.currentNode}
              .completedNodes=${this.completedNodes}
            ></hd-classify-progress>`
          : nothing}
        <div class="meta">
          <span>${formatBytes(doc.size_bytes)}</span>
          <span>${formatDate(doc.uploaded_at)}</span>
        </div>
        <div class="actions">
          <button
            class="btn"
            ?disabled=${doc.status === "complete" || this.classifying}
            @click=${this.handleClassify}
          >Classify</button>
          <button class="btn" @click=${this.handleReview}>Review</button>
        </div>
      </div>
    `;
  }
}
```

## Streaming Orchestration

SSE operations are owned by the **module** closest to the collection concern — not the pure element that triggered the action. The module calls the streaming service, tracks per-item progress via `@state()`, and passes progress data to pure elements as properties.

```typescript
// In the module — owns SSE lifecycle
private handleClassify(e: CustomEvent<{ id: string }>) {
  const docId = e.detail.id;
  if (this.classifying.has(docId)) return;

  const progress: ClassifyProgress = {
    currentNode: null,
    completedNodes: [],
  };

  this.classifying = new Map(this.classifying).set(docId, progress);

  const controller = ClassificationService.classify(docId, {
    onEvent: (type, data) => {
      // Update progress map, trigger re-render
      const updated = new Map(this.classifying);
      // ... handle node.start, node.complete events
      this.classifying = updated;
    },
    onComplete: () => {
      this.abortControllers.delete(docId);
      const updated = new Map(this.classifying);
      updated.delete(docId);
      this.classifying = updated;
      this.fetchDocuments();
    },
    onError: () => {
      // Same cleanup pattern
    },
  });

  this.abortControllers.set(docId, controller);
}
```

The pure element receives all streaming state as properties and dispatches intent events upward. It has no knowledge of services, SSE, or `AbortController`.

## Overlay Elements

Overlay elements use native top-layer primitives — see the SKILL.md "Overlay Convention" section for the decision matrix. Three patterns cover every primitive currently in use.

### Modal dialog (`<dialog>` + `.showModal()`)

```typescript
export type ConfirmKind = "danger" | "primary" | "neutral";

const CONFIRM_CLASS: Record<ConfirmKind, string> = {
  danger: "btn btn-red",
  primary: "btn btn-green",
  neutral: "btn",
};

@customElement("hd-confirm-dialog")
export class ConfirmDialog extends LitElement {
  @property() message = "Are you sure?";
  @property() confirmKind: ConfirmKind = "danger";
  @query("dialog") private dialogEl!: HTMLDialogElement;

  firstUpdated() {
    this.dialogEl.showModal();
  }

  private handleBackdropClick(e: MouseEvent) {
    if (e.target === this.dialogEl) this.emitCancel();
  }

  private handleCancelEvent(e: Event) {
    e.preventDefault();
    this.emitCancel();
  }

  render() {
    return html`
      <dialog @click=${this.handleBackdropClick} @cancel=${this.handleCancelEvent}>
        <div class="panel">
          <p class="message">${this.message}</p>
          <div class="actions">
            <button class="btn" @click=${this.emitCancel}>Cancel</button>
            <button class=${CONFIRM_CLASS[this.confirmKind]} @click=${this.emitConfirm} autofocus>
              Confirm
            </button>
          </div>
        </div>
      </dialog>
    `;
  }
}
```

```css
:host { display: contents; }
dialog { padding: 0; border: 1px solid var(--divider); border-radius: var(--radius-md); }
dialog::backdrop { background: hsl(0 0% 0% / 0.5); }
```

The parent conditionally mounts the dialog and listens for `confirm` / `cancel` events. `<dialog>.showModal()` provides focus trap, Escape → `cancel`, focus return on close, and `::backdrop` styling natively — no manual scrim, no `z-index`. `:host { display: contents }` removes the custom element's own box from the layout tree so mounting the dialog cannot reflow the caller's layout. A semantic-variant enum (`confirmKind`) maps caller intent to a button color class without leaking raw class names — reusable anywhere caller intent should shape presentation.

### Toast stack (`popover="manual"`)

```typescript
@customElement("hd-toast-container")
export class ToastContainer extends LitElement {
  connectedCallback() {
    super.connectedCallback();
    this.setAttribute("popover", "manual");
    this.showPopover();
    this.unsubscribe = Toast.subscribe((t) => { this.toasts = t; });
  }

  disconnectedCallback() {
    this.unsubscribe?.();
    if (this.matches(":popover-open")) this.hidePopover();
    super.disconnectedCallback();
  }
}
```

```css
:host {
  background: transparent;
  border: 0;
  padding: 0;
  margin: 0;
}

.stack {
  position: fixed;
  bottom: var(--space-4);
  left: 0;
  right: 0;
  margin-inline: auto;
  width: min(72ch, calc(100dvw - var(--space-8)));
}
```

`popover="manual"` puts the host in the top layer without light-dismiss — correct when outside-click dismissal would be destructive (would kill every toast on any page click). The `:host` reset neutralizes UA popover chrome (`background: Canvas`, `border: solid`, `padding: 0.25em`) so the popover host is visually inert; the inner `.stack` div owns the fixed positioning because it sits outside the UA `[popover]` cascade and centers predictably.

### Anchored tooltip (`popover="hint"`)

```typescript
@customElement("hd-tooltip")
export class Tooltip extends LitElement {
  @property() message = "";
  @query("span.trigger") private triggerEl!: HTMLSpanElement;
  @query("div.tip") private tipEl!: HTMLDivElement;

  private anchorName = `--hd-tooltip-${++tooltipSeq}`;
  private showTimer: number | undefined;

  firstUpdated() {
    this.triggerEl.style.setProperty("anchor-name", this.anchorName);
    this.tipEl.style.setProperty("position-anchor", this.anchorName);
  }

  render() {
    return html`
      <span class="trigger"
            @mouseenter=${this.handleEnter} @mouseleave=${this.handleLeave}
            @focusin=${this.handleEnter} @focusout=${this.handleLeave}>
        <slot></slot>
      </span>
      <div class="tip" popover="hint" role="tooltip">${this.message}</div>
    `;
  }
}
```

```css
.trigger { display: inline-flex; overflow: hidden; min-width: 0; flex: 1; }
.tip {
  inset: unset;
  top: anchor(bottom);
  justify-self: anchor-center;
  position-try-fallbacks: flip-block;
}
```

`popover="hint"` (not `"auto"`) keeps tooltips from closing open menus while still closing other tooltips. A short show delay (~150ms) avoids flicker on quick mouse travel. `position-try-fallbacks: flip-block` lets the popover flip above when below is tight. The tooltip performs no measurement or gating itself — composing elements conditionally render it (or pass a disabled flag) when they want behavior like "only show when the trigger is truncated."

Anchor positioning uses a per-instance `anchor-name` so multiple tooltip instances don't collide. TypeScript's `lib.dom` does not yet type `CSSStyleDeclaration.anchorName` / `.positionAnchor`, so set them via `setProperty` with the kebab-case CSS names.

## Template Patterns

### Render Methods

Extract complex template logic into private `renderXxx()` methods. Use `nothing` from Lit for conditional non-rendering — it produces no DOM output, unlike an empty string which creates a text node.

```typescript
import { nothing } from "lit";

private renderError() {
  if (!this.error) return nothing;
  return html`<div class="error">${this.error}</div>`;
}

render() {
  return html`
    ${this.renderError()}
    ${this.renderContent()}
  `;
}
```

### Form Handling

Extract values via FormData on submit rather than tracking controlled inputs:

```typescript
private async handleSubmit(e: Event) {
  e.preventDefault();
  const form = e.target as HTMLFormElement;
  const data = new FormData(form);

  const result = await PromptService.create({
    name: data.get("name") as string,
    stage: data.get("stage") as PromptStage,
    instructions: data.get("instructions") as string,
  });

  if (result.ok) {
    this.dispatchEvent(new CustomEvent("prompt-created", {
      bubbles: true, composed: true,
    }));
  }
}

render() {
  return html`
    <form @submit=${this.handleSubmit}>
      <input name="name" required />
      <button type="submit">Save</button>
    </form>
  `;
}
```

### Host Attribute Reflection

Reflect component state to the host element so CSS can drive layout changes without JavaScript:

```typescript
@state() private expanded = false;

updated(changed: Map<string, unknown>) {
  if (changed.has("expanded")) {
    this.toggleAttribute("expanded", this.expanded);
  }
}
```

```css
:host { grid-template-rows: auto auto 1fr; }
:host([expanded]) { grid-template-rows: auto 1fr 1fr; }
```

### Object URL Lifecycle

Revoke blob URLs in `disconnectedCallback` to prevent memory leaks:

```typescript
private imageUrls = new Map<File, string>();

disconnectedCallback() {
  super.disconnectedCallback();
  this.imageUrls.forEach((url) => URL.revokeObjectURL(url));
  this.imageUrls.clear();
}

private getImageUrl(file: File): string {
  let url = this.imageUrls.get(file);
  if (!url) {
    url = URL.createObjectURL(file);
    this.imageUrls.set(file, url);
  }
  return url;
}
```
