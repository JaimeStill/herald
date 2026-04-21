# 133 - Scroll container for document-upload module

## Problem Context

When many files are queued in the `hd-document-upload` module, the queue container grows without bounds and overflows the viewport because no scroll container is declared. The sibling `document-grid` and `prompt-list` modules already handle this cleanly.

The module sits inside `documents-view.module.css`'s `.view` flex column (`flex: 1; min-height: 0;`). Its own `:host` currently has no flex sizing, so it expands to intrinsic content height — pushing `<hd-document-grid>` out of view and letting the queue list grow past the viewport instead of scrolling internally.

## Architecture Approach

Mirror the overflow pattern used by `prompt-list.module.css` and `document-grid.module.css`, with one refinement: the File Queue header (title, file count, Clear/Upload actions) stays pinned — only the entries scroll.

- Host becomes a flex-growing, min-height-0 column so the module participates correctly in its parent flex layout.
- The fixed-height sibling (`.drop-zone`) is pinned above the queue with explicit `flex-shrink: 0;`.
- `.queue` remains a flex column that claims vertical space (`flex: 1; min-height: 0;`), but is no longer the scroll container.
- `.queue-header` is pinned inside `.queue` with `flex-shrink: 0;`.
- A new `.queue-list` wrapper around the mapped entries becomes the internal scroll container.

Requires a small markup change in `document-upload.ts` (wrap the entries in `.queue-list`) plus the CSS updates.

## Implementation

### Step 1: Update `app/client/ui/modules/document-upload.module.css`

Show only the modified rules; other declarations in each rule stay as-is.

**`:host`** — add `flex: 1; min-height: 0;`:

```css
:host {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
  flex: 1;
  min-height: 0;
}
```

**`.drop-zone`** — add `flex-shrink: 0;` to keep it pinned above the queue:

```css
.drop-zone {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: var(--space-2);
  padding: var(--space-8);
  border: 2px dashed var(--divider);
  border-radius: var(--radius-md);
  cursor: pointer;
  flex-shrink: 0;
  transition:
    border-color 0.15s,
    background 0.15s;

  &:hover {
    border-color: var(--color-2);
    background: var(--bg-1);
  }
}
```

**`.queue`** — add `flex: 1; min-height: 0;` so it claims vertical space, but do **not** set `overflow-y`:

```css
.queue {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  border: 1px solid var(--divider);
  border-radius: var(--radius-md);
  padding: var(--space-3);
  background: var(--bg-1);
  flex: 1;
  min-height: 0;
}
```

**`.queue-header`** — add `flex-shrink: 0;` so it stays pinned when the list scrolls:

```css
.queue-header {
  display: grid;
  grid-template-columns: 1fr auto 1fr;
  align-items: center;
  padding-bottom: var(--space-2);
  border-bottom: 1px solid var(--divider);
  flex-shrink: 0;
}
```

**New `.queue-list` rule** — the actual scroll container for queue entries. Place it directly below the `.queue-header` rule:

```css
.queue-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  flex: 1;
  min-height: 0;
  overflow-y: auto;
}
```

### Step 2: Wrap queue entries in `.queue-list` in `app/client/ui/modules/document-upload.ts`

Update `renderQueue()` so the mapped entries live inside a dedicated scroll wrapper (around line 258). Existing:

```ts
  private renderQueue() {
    if (this.queue.length < 1) return nothing;

    return html`
      <div class="queue">
        <div class="queue-header">
          <span class="queue-title">File Queue</span>
          <span class="queue-count">
            ${this.queue.length} file${this.queue.length > 1 ? "s" : ""}
          </span>
          ${this.renderQueueActions()}
        </div>
        ${this.queue.map((entry, i) => this.renderQueueEntry(entry, i))}
      </div>
    `;
  }
```

Replace the body with:

```ts
  private renderQueue() {
    if (this.queue.length < 1) return nothing;

    return html`
      <div class="queue">
        <div class="queue-header">
          <span class="queue-title">File Queue</span>
          <span class="queue-count">
            ${this.queue.length} file${this.queue.length > 1 ? "s" : ""}
          </span>
          ${this.renderQueueActions()}
        </div>
        <div class="queue-list">
          ${this.queue.map((entry, i) => this.renderQueueEntry(entry, i))}
        </div>
      </div>
    `;
  }
```

## Remediation

### R1: Scrollbar Gutter Design System Primitive

**Context.** With the new `.queue-list` scroll container introduced above, the app now has seven scroll containers (one added here, six pre-existing). Each reserves its own scrollbar behavior ad-hoc via `overflow-y: auto`, with no shared gutter reservation or visual separation between content and scrollbar. This remediation extracts a reusable primitive so scroll behavior is consistent across every module and so future scroll containers inherit the pattern by convention.

**Shadow DOM constraint.** Components are Lit elements; styles in `design/index.css` do not pierce shadow roots. The design system therefore needs two tiers: a light-DOM rule for document-level scroll contexts and a component-adoptable CSS module for shadow roots.

**Pattern overview.**

- **Light DOM** — new `base` layer between `reset` and `theme` with a universal `scrollbar-gutter: stable` rule. `scrollbar-gutter` is a no-op on elements whose `overflow` is not `auto`/`scroll`/`hidden`, so the `*` selector is safe.
- **Shadow DOM** — new `scroll.module.css` exposing `.scroll-y` and `.scroll-x` utilities that components import via the `@styles/` alias (same pattern as `buttons.module.css`, `inputs.module.css`). Each utility bundles `overflow-*: auto`, `scrollbar-gutter: stable`, and a `padding-*` axis to keep the scrollbar visually separated from content.

The refactor swaps every per-component `overflow-y: auto` declaration for the `.scroll-y` utility class applied in the template.

### R1 Step 1: Add `app/client/design/core/base.css`

New file:

```css
@layer base {
  * {
    scrollbar-gutter: stable;
  }
}
```

### R1 Step 2: Register the `base` layer in `app/client/design/index.css`

Insert `base` into the layer order (between `reset` and `theme`) and add the import:

```css
@layer tokens, reset, base, theme, app;

@import url(./core/tokens.css);
@import url(./core/reset.css);
@import url(./core/base.css);
@import url(./core/theme.css);

@import url(./app/app.css);
```

### R1 Step 3: Add `app/client/design/styles/scroll.module.css`

New file. `padding-inline-end` / `padding-block-end` reserves visible space so the scrollbar does not abut content:

```css
.scroll-y {
  overflow-y: auto;
  scrollbar-gutter: stable;
  padding-inline-end: var(--space-2);
}

.scroll-x {
  overflow-x: auto;
  scrollbar-gutter: stable;
  padding-block-end: var(--space-2);
}
```

### R1 Step 4: Migrate existing scroll containers to `.scroll-y`

For every scroll container, do the following three things:

1. In the `.module.css`, **remove** the `overflow-y: auto` declaration from the rule. Leave any surrounding layout declarations (`flex: 1; min-height: 0;`, `max-height: ...`, etc.) in place.
2. In the component `.ts` file, add the scroll styles import and include it in `static styles`:

   ```ts
   import scrollStyles from "@styles/scroll.module.css";
   // ...
   static styles = [/* existing */, scrollStyles, styles];
   ```

3. In the component template, add `scroll-y` to the scroll container's `class` attribute.

**Touchpoints:**

| Component | File (module.css → ts) | Selector losing `overflow-y: auto` | Template element to add `scroll-y` to |
|-----------|------------------------|------------------------------------|---------------------------------------|
| document-grid | `ui/modules/document-grid.*` | `.grid` | `<div class="grid">` → `<div class="grid scroll-y">` |
| prompt-list | `ui/modules/prompt-list.*` | `.list` | `<div class="list">` → `<div class="list scroll-y">` |
| classification-panel | `ui/modules/classification-panel.*` | `.panel-body` | All three `.panel-body` usages (lines ~165, ~215, ~257) |
| prompt-form | `ui/modules/prompt-form.*` | `.form-body` | `<form class="form-body">` → `<form class="form-body scroll-y">` |
| prompt-form (inner) | `ui/modules/prompt-form.*` | `.defaults-content` (keep `max-height: 20rem;`) | `<div class="defaults-content">` → `<div class="defaults-content scroll-y">` |
| review-view | `ui/views/review-view.*` | `.classification-panel` | `<div class="panel classification-panel">` → `<div class="panel classification-panel scroll-y">` |
| document-upload | `ui/modules/document-upload.*` (from Step 1 above) | `.queue-list` | `<div class="queue-list">` → `<div class="queue-list scroll-y">` |

For `document-upload`, remove the `overflow-y: auto;` from the `.queue-list` rule added in Step 1 above so it matches the refactored pattern — the utility class now owns that declaration.

### R1 Step 5: Ensure `document-upload.ts` imports `scrollStyles`

Update the import block and `static styles` in `app/client/ui/modules/document-upload.ts`:

```ts
import badgeStyles from "@styles/badge.module.css";
import buttonStyles from "@styles/buttons.module.css";
import scrollStyles from "@styles/scroll.module.css";
import styles from "./document-upload.module.css";
```

```ts
static styles = [buttonStyles, badgeStyles, scrollStyles, styles];
```

## Validation Criteria

- [ ] CSS updates applied to `:host`, `.drop-zone`, `.queue`, `.queue-header`, and new `.queue-list`
- [ ] `renderQueue()` wraps entries in `<div class="queue-list scroll-y">`
- [ ] `design/core/base.css` exists and is imported in `design/index.css` under the `base` layer
- [ ] `design/styles/scroll.module.css` exists with `.scroll-y` and `.scroll-x` utilities
- [ ] All seven scroll containers listed in R1 Step 4 have their local `overflow-y: auto` removed and the `scroll-y` class applied, with `scrollStyles` imported into each component's `static styles`
- [ ] `mise run dev` starts cleanly
- [ ] Navigate to `/documents`, click **Upload**, drop ≥ 20 PDFs — only the entries scroll; the "File Queue" title, count, and Clear/Upload buttons stay pinned at the top of the queue
- [ ] Scrollbar has visible breathing room from the rightmost content in every scroll container (queue list, document grid, prompt list, classification panel, prompt form body and defaults, review-view classification panel)
- [ ] No horizontal layout shift occurs when a container's content grows past the fold (scrollbar-gutter reserved consistently)
- [ ] Drop zone remains visible above the queue regardless of entry count
- [ ] Drag-and-drop works over the drop zone with both empty and populated queues
- [ ] During upload, progress rows and completion badges render correctly without layout shift
- [ ] Closing the upload panel leaves the document grid rendering normally
- [ ] `mise run vet` passes
