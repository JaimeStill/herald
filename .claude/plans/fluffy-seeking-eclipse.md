# Plan: Adopt native overlay primitives (#145)

## Context

Herald's overlay UI today is a patchwork. `hd-confirm-dialog` is a manual `position: fixed; z-index: 100` overlay with no focus trap, no Escape handler, and manual backdrop wiring. `hd-toast-container` uses `z-index: 200`, which works **only** until `hd-confirm-dialog` moves to the top layer — at that point toasts render behind open modals. Truncated filenames on `hd-document-card` and prompt names on `hd-prompt-card` silently hide content with no recourse because there is no tooltip primitive.

Native top-layer primitives (`<dialog>`, Popover API) and CSS Anchor Positioning are baseline in modern browsers and solve all three problems cleanly. This task adopts them consistently and establishes the convention documentation for future overlay work.

The convention text already landed in `.claude/CLAUDE.md` (line 99) and `.claude/skills/web-development/SKILL.md` (lines 94–127) during prior Phase 5 work. What is still missing: (a) the code patterns in `references/components.md`, and (b) the actual refactored elements.

## Implementation

### Step 1 — New element: `hd-tooltip`

**Create** `app/client/ui/elements/tooltip.ts` and `app/client/ui/elements/tooltip.module.css`.

Element responsibilities:
- `@property({ type: String }) message` — text to render.
- `<slot>` wraps the trigger verbatim so host layout is untouched.
- Internal template renders a `<span>` trigger wrapper and a sibling `<div popover="hint">` with `anchor-name` bound to the wrapper.
- Truncation detection: in `firstUpdated` and on `ResizeObserver` callback, compare `trigger.scrollWidth > trigger.clientWidth` on the first assignable child of the slot. When false, remove listeners and become inert.
- Show logic: `mouseenter`/`focusin` → schedule a 150ms timer → `popover.showPopover()`. `mouseleave`/`focusout` → cancel timer and `popover.hidePopover()` guarded by `:popover-open`.
- Teardown: disconnect `ResizeObserver` and clear pending timer in `disconnectedCallback`.
- Uses `popover="hint"` (not `auto`) so hovering a tooltip inside an open menu does not close the menu.

CSS (`tooltip.module.css`):
- `[popover]` tooltip: monospace, `--bg-2` background, `--shadow-md`, `--text-xs`, no default margin, `border-radius: var(--radius-sm)`, padding `var(--space-2) var(--space-3)`.
- Trigger wrapper gets `anchor-name: --hd-tooltip-<id>` (generate a unique id per instance; a simple counter module variable is fine).
- Popover positioning via `position-anchor: --hd-tooltip-<id>`, `top: anchor(bottom)` with `margin-top: var(--space-1)`, and `position-try-fallbacks: flip-block` so it flips above when below lacks room.
- Clear UA popover defaults (`inset: unset`, `margin: 0`) so anchor positioning is authoritative.

Register in `app/client/ui/elements/index.ts` — add `export { Tooltip } from "./tooltip";`.

### Step 2 — Refactor `hd-confirm-dialog` to `<dialog>`

**Edit** `app/client/ui/elements/confirm-dialog.ts`:
- Replace the `<div class="overlay"><div class="dialog">` template with a single `<dialog>` element.
- Add `@query("dialog") private dialogEl!: HTMLDialogElement`.
- `firstUpdated()` calls `this.dialogEl.showModal()`.
- Bind `@click` on the `<dialog>` to a handler that emits `cancel` when `event.target === this.dialogEl` (backdrop click).
- Bind `@cancel` on the `<dialog>` (native Escape event) to call `event.preventDefault()` and dispatch the existing `cancel` `CustomEvent`.
- Keep the existing `handleConfirm` / `handleCancel` methods; they continue dispatching the same events. Drop the manual `stopPropagation` on the inner wrapper.
- Primary action button: add `autofocus` attribute on the Confirm button so keyboard users can confirm immediately.

**Edit** `app/client/ui/elements/confirm-dialog.module.css`:
- Delete the `.overlay` rule entirely.
- Move the `.dialog` panel styles onto the bare `dialog` selector, dropping `z-index`.
- Reset UA dialog chrome: `dialog { padding: 0; border: 1px solid var(--divider); background: var(--bg-1); border-radius: var(--radius-md); }` and keep the existing panel padding on an inner wrapper or just apply padding directly to the `dialog`.
- Add `dialog::backdrop { background: hsl(0 0% 0% / 0.5); }`.
- No `z-index` anywhere.

Caller contract unchanged: `document-grid.ts` and `prompt-list.ts` continue to render `<hd-confirm-dialog message="..." @confirm=${...} @cancel=${...}>` conditionally.

### Step 3 — Refactor `hd-toast-container` to `popover="manual"`

**Edit** `app/client/ui/elements/toast.ts`:
- In `connectedCallback`, after `super.connectedCallback()`, call `this.setAttribute("popover", "manual")` and `this.showPopover()`. (Using setAttribute avoids needing a reflected property and keeps the existing `@state` surface clean.)
- In `disconnectedCallback`, guard with `if (this.matches(":popover-open")) this.hidePopover()` before removing the subscription, then call super.
- No other logic changes — the Toast service, bus, timers, and render method stay identical.

**Edit** `app/client/ui/elements/toast.module.css`:
- Remove the `.stack` `position: fixed`, `bottom`, `left`, `right`, `margin-inline`, `z-index`, and `pointer-events` declarations. Move them onto `:host` since the host is now the top-layer popover.
- On `:host`, set: `position: fixed; bottom: var(--space-4); left: 0; right: 0; margin-inline: auto; width: min(72ch, calc(100dvw - var(--space-8))); pointer-events: none;` — keep the `.stack` div as the flex column inside the host.
- UA popover defaults: add `:host { margin: 0; padding: 0; border: 0; background: transparent; overflow: visible; inset: unset; }` so the browser's default popover positioning does not fight our fixed layout.
- `.stack` retains `display: flex; flex-direction: column; gap: var(--space-2);`.
- `.toast` retains `pointer-events: auto` and everything else.

### Step 4 — Wrap card triggers with tooltips

**Edit** `app/client/ui/elements/document-card.ts`:
- Import the Tooltip element side-effect (the barrel `elements/index.ts` registration is sufficient since `document-card.ts` is already registered the same way, but because tooltip lives in the same directory and both import from `lit`, simply importing `./tooltip` at the top of `document-card.ts` is cleanest). Alternative: no import needed if `elements/index.ts` is loaded before views — verify. Pick the direct import approach for explicitness.
- Change `<span class="filename">${doc.filename}</span>` to:
  ```html
  <hd-tooltip message=${doc.filename}>
    <span class="filename">${doc.filename}</span>
  </hd-tooltip>
  ```

**Edit** `app/client/ui/elements/prompt-card.ts`:
- Same pattern for the `<span class="name">${p.name}</span>` in the header.

No CSS changes on the cards — the tooltip slots the existing `.filename`/`.name` span with its existing ellipsis rules intact.

### Step 5 — Document the patterns in `references/components.md`

**Edit** `.claude/skills/web-development/references/components.md`:

Insert a new top-level section **"Overlay Elements"** between `## Streaming Orchestration` (ends ~line 278) and `## Template Patterns` (line 280). Include three subsections:

1. `### Modal dialog (<dialog> + .showModal())` — code excerpt from the refactored `hd-confirm-dialog`: query the dialog, call `.showModal()` in `firstUpdated`, wire `cancel` and backdrop click, `::backdrop` styling, drop z-index.
2. `### Toast stack (popover="manual")` — connectedCallback pattern (`setAttribute("popover", "manual")` then `showPopover()`) and disconnectedCallback guard.
3. `### Anchored tooltip (popover="hint")` — element structure, anchor-name wiring, `ResizeObserver`-driven truncation detection, 150ms show delay.

Keep each example short — tight excerpts, not full file dumps.

## File-change summary

| File | Action |
|------|--------|
| `app/client/ui/elements/tooltip.ts` | Create |
| `app/client/ui/elements/tooltip.module.css` | Create |
| `app/client/ui/elements/index.ts` | Export Tooltip |
| `app/client/ui/elements/confirm-dialog.ts` | Refactor to `<dialog>` |
| `app/client/ui/elements/confirm-dialog.module.css` | Drop overlay; add `::backdrop` |
| `app/client/ui/elements/toast.ts` | Popover=manual lifecycle |
| `app/client/ui/elements/toast.module.css` | Move fixed/layout from `.stack` to `:host` |
| `app/client/ui/elements/document-card.ts` | Wrap filename span with `<hd-tooltip>` |
| `app/client/ui/elements/prompt-card.ts` | Wrap name span with `<hd-tooltip>` |
| `.claude/skills/web-development/references/components.md` | Add "Overlay Elements" section |

No changes needed to `.claude/CLAUDE.md` or `SKILL.md` — overlay convention text is already present from earlier Phase 5 work.

## Verification

1. `bun run build` succeeds with no type errors.
2. Start `mise run dev` and `bun run watch`; exercise each overlay in the browser:
   - Single delete on a document card → confirm dialog opens via top layer, Escape fires cancel, backdrop click fires cancel, Enter on the focused Confirm button deletes.
   - Bulk delete → same semantics.
   - Prompt delete → same semantics.
   - Hover a document card with a truncated filename → tooltip appears above after ~150ms; quick hover-out before 150ms produces no flicker. Tooltip flips below when the card sits near the top of the viewport.
   - Hover a card whose filename fits → no tooltip.
   - Resize the viewport so a previously fitting filename now truncates → next hover shows the tooltip (ResizeObserver re-evaluates).
   - Trigger a background classify that completes while a delete modal is open → the success toast renders **above** the modal backdrop (stacking regression test).
   - Toast still auto-dismisses, still dismisses on click, still caps at 5 visible.
3. Accessibility spot-check: Tab into an open `hd-confirm-dialog` → focus stays inside the dialog (native focus trap); close it → focus returns to the triggering button.
4. DOM inspection confirms no `z-index` is set on any overlay element (`document.querySelector('hd-toast-container').style.zIndex === ''`).

Done when all validation checkboxes in the issue body pass.
