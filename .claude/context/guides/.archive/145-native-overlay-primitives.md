# 145 - Adopt native overlay primitives for tooltips, modals, and toasts

## Problem Context

Herald's overlay UI today is a patchwork:

- `hd-confirm-dialog` is a manual overlay div (`position: fixed; inset: 0; z-index: 100`) with no focus trap, no Escape handler, no autofocus management, no `role="dialog"` / `aria-modal`, and manual backdrop-click wiring via `stopPropagation`.
- There is no tooltip capability, so truncated filenames (`document-card`) and prompt names (`prompt-card`) silently hide content from the user.
- `hd-toast-container` uses `z-index: 200`, which works today but would render **behind** any `<dialog>.showModal()` the moment we move confirm-dialog to the top layer.

Native browser primitives (`<dialog>`, Popover API, CSS Anchor Positioning) are baseline in 2026 and solve all three cleanly. This task adopts them consistently and establishes the convention for all future overlay UI.

## Architecture Approach

Use the browser's top layer as the single source of stacking truth. No `z-index` on any overlay element. Three primitives map one-to-one to our three overlay classes:

| Overlay | Primitive | Lifecycle owner |
|--------|-----------|-----------------|
| `hd-confirm-dialog` | `<dialog>.showModal()` | Element `firstUpdated` opens; `close()` on confirm |
| `hd-toast-container` | `popover="manual"` | `connectedCallback` shows; `disconnectedCallback` hides |
| `hd-tooltip` (new) | `popover="hint"` | `mouseenter`/`focusin` show; `mouseleave`/`focusout` hide |

`popover="hint"` (not `"auto"`) on the tooltip is deliberate: hints do not close open `auto` popovers, so hovering a tooltip inside an open menu leaves the menu open. Hints do close other hints, so only one tooltip is ever visible.

The convention text is already present in `.claude/CLAUDE.md` and `.claude/skills/web-development/SKILL.md`; this task adds concrete code examples to `references/components.md`.

## Implementation

### Step 1: Create `hd-tooltip`

**New file:** `app/client/ui/elements/tooltip.ts`

```typescript
import { LitElement, html } from "lit";
import { customElement, property, query } from "lit/decorators.js";

import styles from "./tooltip.module.css";

let tooltipSeq = 0;

const SHOW_DELAY_MS = 150;

@customElement("hd-tooltip")
export class Tooltip extends LitElement {
  static styles = [styles];

  @property() message = "";

  @query("span.trigger") private triggerEl!: HTMLSpanElement;
  @query("div.tip") private tipEl!: HTMLDivElement;

  private anchorName = `--hd-tooltip-${++tooltipSeq}`;
  private showTimer: number | undefined;

  firstUpdated() {
    this.triggerEl.style.setProperty("anchor-name", this.anchorName);
    this.tipEl.style.setProperty("position-anchor", this.anchorName);
  }

  disconnectedCallback() {
    super.disconnectedCallback();
    this.clearShowTimer();
    if (this.tipEl?.matches(":popover-open")) this.tipEl.hidePopover();
  }

  private handleEnter = () => {
    this.clearShowTimer();
    this.showTimer = window.setTimeout(() => {
      if (!this.tipEl.matches(":popover-open")) this.tipEl.showPopover();
    }, SHOW_DELAY_MS);
  };

  private handleLeave = () => {
    this.clearShowTimer();
    if (this.tipEl?.matches(":popover-open")) this.tipEl.hidePopover();
  };

  private clearShowTimer() {
    if (this.showTimer !== undefined) {
      clearTimeout(this.showTimer);
      this.showTimer = undefined;
    }
  }

  render() {
    return html`
      <span
        class="trigger"
        @mouseenter=${this.handleEnter}
        @mouseleave=${this.handleLeave}
        @focusin=${this.handleEnter}
        @focusout=${this.handleLeave}
      >
        <slot></slot>
      </span>
      <div class="tip" popover="hint" role="tooltip">${this.message}</div>
    `;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    "hd-tooltip": Tooltip;
  }
}
```

The tooltip is a dumb primitive — it always shows on hover/focus (with a 150ms delay). Any gating behavior (e.g., "only show when the trigger content is truncated") is the composing element's responsibility; the tooltip itself performs no measurement.

**New file:** `app/client/ui/elements/tooltip.module.css`

```css
:host {
  display: contents;
}

.trigger {
  display: inline-flex;
  min-width: 0;
  flex: 1;
  overflow: hidden;
}

.trigger ::slotted(*) {
  min-width: 0;
  max-width: 100%;
}

.tip {
  margin: 0;
  inset: unset;
  padding: var(--space-2) var(--space-3);
  background: var(--bg-2);
  color: var(--color);
  border: 1px solid var(--divider);
  border-radius: var(--radius-sm);
  box-shadow: var(--shadow-md);
  font-family: var(--font-mono);
  font-size: var(--text-xs);
  max-width: min(40ch, 90vw);
  white-space: normal;
  overflow-wrap: break-word;
  top: anchor(bottom);
  justify-self: anchor-center;
  margin-top: var(--space-1);
  position-try-fallbacks: flip-block;
}
```

**Edit:** `app/client/ui/elements/index.ts`

Add the new `Tooltip` export and a `ConfirmKind` type re-export (for callers that want type-safe `confirmKind` values):

```typescript
export { BlobViewer } from "./blob-viewer";
export { ClassifyProgress } from "./classify-progress";
export { ConfirmDialog } from "./confirm-dialog";
export type { ConfirmKind } from "./confirm-dialog";
export { DocumentCard } from "./document-card";
export { MarkingsList } from "./markings-list";
export { PaginationControls } from "./pagination-controls";
export { PromptCard } from "./prompt-card";
export { Toast, ToastContainer } from "./toast";
export type { ToastItem, ToastKind } from "./toast";
export { Tooltip } from "./tooltip";
```

### Step 2: Refactor `hd-confirm-dialog` to `<dialog>`

**Edit:** `app/client/ui/elements/confirm-dialog.ts`

Replace the entire file body (keep the file, replace contents) with:

```typescript
import { LitElement, html } from "lit";
import { customElement, property, query } from "lit/decorators.js";

import buttonStyles from "@styles/buttons.module.css";
import styles from "./confirm-dialog.module.css";

export type ConfirmKind = "danger" | "primary" | "neutral";

const CONFIRM_CLASS: Record<ConfirmKind, string> = {
  danger: "btn btn-red",
  primary: "btn btn-green",
  neutral: "btn",
};

@customElement("hd-confirm-dialog")
export class ConfirmDialog extends LitElement {
  static styles = [buttonStyles, styles];

  @property() message = "Are you sure?";
  @property() confirmKind: ConfirmKind = "danger";

  @query("dialog") private dialogEl!: HTMLDialogElement;

  firstUpdated() {
    this.dialogEl.showModal();
  }

  private handleConfirm() {
    this.dispatchEvent(
      new CustomEvent("confirm", { bubbles: true, composed: true }),
    );
    this.dialogEl.close();
  }

  private handleCancel() {
    this.dispatchEvent(
      new CustomEvent("cancel", { bubbles: true, composed: true }),
    );
  }

  private handleBackdropClick(e: MouseEvent) {
    if (e.target === this.dialogEl) this.handleCancel();
  }

  private handleCancelEvent(e: Event) {
    e.preventDefault();
    this.handleCancel();
  }

  render() {
    return html`
      <dialog
        @click=${this.handleBackdropClick}
        @cancel=${this.handleCancelEvent}
      >
        <div class="panel">
          <p class="message">${this.message}</p>
          <div class="actions">
            <button class="btn" @click=${this.handleCancel}>Cancel</button>
            <button
              class=${CONFIRM_CLASS[this.confirmKind]}
              @click=${this.handleConfirm}
              autofocus
            >
              Confirm
            </button>
          </div>
        </div>
      </dialog>
    `;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    "hd-confirm-dialog": ConfirmDialog;
  }
}
```

Default is `"danger"` so the three existing callers (bulk delete, single delete, prompt delete) need no changes. Future callers set `confirmKind="primary"` for affirmative actions or `confirmKind="neutral"` for mild confirmations. The enum keeps the styling system encapsulated — callers don't pass raw CSS class names.

**Edit:** `app/client/ui/elements/confirm-dialog.module.css`

Replace the entire file body with:

```css
:host {
  display: contents;
}

dialog {
  padding: 0;
  background: var(--bg-1);
  color: var(--color);
  border: 1px solid var(--divider);
  border-radius: var(--radius-md);
  max-width: 24rem;
  width: 100%;
}

dialog::backdrop {
  background: hsl(0 0% 0% / 0.5);
}

.panel {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
  padding: var(--space-6);
}

.message {
  font-size: var(--text-sm);
  color: var(--color);
  margin: 0;
}

.actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-2);
}
```

`:host { display: contents }` removes the `<hd-confirm-dialog>` host's own box from the layout tree. Without it the host is a default `display: inline` custom element that, when inserted into the document-grid's template, contributes enough layout baseline to reflow the grid rows and push the pagination row past the scroll boundary. With `display: contents` only the shadow DOM's `<dialog>` (top layer) and `::backdrop` render — the host is invisible to flex/grid calculations and the underlying view stays still when the dialog opens.

### Step 3: Refactor `hd-toast-container` to `popover="manual"`

**Edit:** `app/client/ui/elements/toast.ts`

Only the `ToastContainer` class changes; the `ToastBus`, `Toast` service, helpers, and render remain identical. Update `connectedCallback` and `disconnectedCallback`:

```typescript
connectedCallback() {
  super.connectedCallback();
  this.setAttribute("popover", "manual");
  this.showPopover();
  this.unsubscribe = Toast.subscribe((toasts) => {
    this.toasts = toasts;
  });
}

disconnectedCallback() {
  this.unsubscribe?.();
  this.unsubscribe = undefined;
  if (this.matches(":popover-open")) this.hidePopover();
  super.disconnectedCallback();
}
```

No other changes inside this file.

**Edit:** `app/client/ui/elements/toast.module.css`

Add a `:host` reset at the top and keep the original positioning on `.stack`. The only reason `:host` appears at all is to neutralize UA popover chrome (`background: Canvas`, `border: solid`, `padding: 0.25em`) so the host's 0×0 box leaves no visible trace. The positioning stays where it was before the refactor because `.stack` is a plain div that isn't in the UA `[popover]` cascade — far fewer surprises.

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
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  pointer-events: none;
}
```

Keep every `.toast`, `.toast:hover`, `.toast.success`, `.toast.error`, `.toast.warning`, `.toast.info`, and `@keyframes slide-in` rule below unchanged. The `z-index: 200` from the old `.stack` is gone — top layer via `popover="manual"` handles stacking.

### Step 4: Wrap card triggers with tooltips

**Edit:** `app/client/ui/elements/document-card.ts`

Add a side-effect import so the `hd-tooltip` element is registered when `hd-document-card` is loaded. Place it alphabetically with the other relative imports:

```typescript
import "./tooltip";
```

In the `render()` method, replace the filename span with a tooltip-wrapped version:

```typescript
<hd-tooltip .message=${doc.filename}>
  <span class="filename">${doc.filename}</span>
</hd-tooltip>
```

No CSS changes.

**Edit:** `app/client/ui/elements/prompt-card.ts`

Add the same side-effect import:

```typescript
import "./tooltip";
```

In `render()`, replace the name span:

```typescript
<hd-tooltip .message=${p.name}>
  <span class="name">${p.name}</span>
</hd-tooltip>
```

No CSS changes.

### Step 5: Scope `scrollbar-gutter: stable` to scroll containers only

**Context:** Issue #133 introduced `* { scrollbar-gutter: stable }` in `app/client/design/core/base.css` on the premise that the rule is a no-op on non-scroll containers. That premise was wrong: `scrollbar-gutter: stable` reserves gutter space on any element whose `overflow` is `auto`, `scroll`, **or `hidden`**. Herald's app shell uses `overflow: hidden` on the body and several non-scrolling flex containers, so the universal rule produces a visible ~15px phantom gutter along the right edge of the layout.

The real scroll containers — `.scroll-y` / `.scroll-x` in `app/client/design/styles/scroll.module.css` — already include `scrollbar-gutter: stable` themselves, so the universal rule is redundant on top of being harmful.

**Edit:** `app/client/design/core/base.css`

Delete the file. It currently contains only the universal rule and nothing else lives in the `base` layer.

**Edit:** `app/client/design/index.css`

Remove the `@import url(./core/base.css);` line (line 5). Leave the `@layer tokens, reset, base, theme, app;` declaration on line 1 untouched so the `base` layer remains available if a future cross-cutting primitive needs it.

**Verify:** `app/client/design/styles/scroll.module.css` — both `.scroll-y` and `.scroll-x` already include `scrollbar-gutter: stable`. No change needed.

## Validation Criteria

### hd-tooltip
- [ ] `<hd-tooltip>` renders no visible chrome by default (pure wrapper)
- [ ] Hovering the trigger shows the tooltip after ~150ms
- [ ] Focusing the trigger (Tab) shows the tooltip
- [ ] Leaving hover/focus hides the tooltip; quick in/out does not flash it
- [ ] Tooltip flips above/below via `position-try-fallbacks`
- [ ] Applied to `document-card` filename and `prompt-card` name

### hd-confirm-dialog
- [ ] Dialog opens via `.showModal()` — top layer
- [ ] `::backdrop` supplies the dim overlay; no overlay div exists
- [ ] Escape fires the `cancel` CustomEvent
- [ ] Clicking the backdrop fires the `cancel` CustomEvent
- [ ] Focus is trapped inside the dialog (Tab cycles within)
- [ ] Confirm button has `autofocus`; focus returns to the trigger on close
- [ ] No `z-index` set on any dialog part
- [ ] All callers (bulk delete, single delete, prompt delete) work unchanged

### hd-toast-container
- [ ] Host has `popover="manual"` and is shown via `.showPopover()` in `connectedCallback`
- [ ] Toasts render above an open `hd-confirm-dialog` modal
- [ ] Toast stack is bottom-centered in the viewport
- [ ] No `z-index` anywhere in `toast.module.css`
- [ ] Click-to-dismiss still works
- [ ] Auto-dismiss timers still work

### scrollbar-gutter scope
- [ ] `app/client/design/core/base.css` no longer exists (or no longer contains the universal rule)
- [ ] `app/client/design/index.css` no longer imports `./core/base.css`
- [ ] No phantom 15px gutter appears on the right edge of the app shell
- [ ] `.scroll-y` / `.scroll-x` containers still reserve gutter and do not layout-shift when content crosses the scroll threshold
