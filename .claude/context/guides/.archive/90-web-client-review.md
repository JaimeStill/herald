# 90 — Web Client Review & Optimization

## Context

Final task in Objective 5 (Document Review View). The review view composition was completed as a remediation in #89. This guide captures source code optimizations identified during a holistic review of the web client architecture.

## 1. Shared Styles — New CSS Modules

Extract duplicated CSS patterns into `app/client/design/styles/`.

### `inputs.module.css`

Form input base styles shared across toolbar inputs, form fields, and upload fields.

```css
.input {
  padding: var(--space-2) var(--space-3);
  border: 1px solid var(--divider);
  border-radius: var(--radius-sm);
  background: var(--bg-1);
  color: var(--color);
  font-size: var(--text-sm);
  font-family: var(--font-mono);

  &:focus-visible {
    outline: 2px solid var(--blue);
    outline-offset: 2px;
  }

  &:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }
}
```

Consumers add the `.input` class to `<input>`, `<select>`, and `<textarea>` elements. Component CSS retains layout-specific overrides (e.g., `.search-input { flex: 1; min-width: 12rem; }`).

### `labels.module.css`

Section label typography for form labels and section headers.

```css
.label {
  font-size: var(--text-xs);
  font-family: var(--font-mono);
  color: var(--color-1);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}
```

Consumers add the `.label` class to `<label>`, `<span>`, and heading elements that serve as section labels.

### `cards.module.css`

Card container pattern shared by document-card and prompt-card.

```css
.card {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  padding: var(--space-4);
  background: var(--bg-1);
  border: 1px solid var(--divider);
  border-radius: var(--radius-md);
  transition: border-color 0.15s;
}
```

### Button color variants in `buttons.module.css`

Add semantic color variants to the existing `.btn` base:

```css
.btn-blue:not(:disabled) {
  border-color: var(--blue);
  color: var(--blue);
  &:hover { background: var(--blue-bg); }
}

.btn-green:not(:disabled) {
  border-color: var(--green);
  color: var(--green);
  &:hover { background: var(--green-bg); }
}

.btn-red:not(:disabled) {
  border-color: var(--red);
  color: var(--red);
  &:hover { background: var(--red-bg); }
}

.btn-yellow:not(:disabled) {
  border-color: var(--yellow);
  color: var(--yellow);
  &:hover { background: var(--yellow-bg); }
}

.btn-muted:not(:disabled) {
  border-color: var(--color-2);
  color: var(--color-2);
  &:hover { background: var(--bg-2); }
}
```

## 2. Component CSS — Consume Shared Styles

For each component, import the relevant shared styles into `static styles`, add shared classes to template markup, and remove the duplicated CSS rules from the component's `.module.css`.

---

### `document-card.ts`

**Imports** — add `cardStyles`:

```typescript
import badgeStyles from "@styles/badge.module.css";
import buttonStyles from "@styles/buttons.module.css";
import cardStyles from "@styles/cards.module.css";
import styles from "./document-card.module.css";
```

**Static styles:**

```typescript
static styles = [buttonStyles, badgeStyles, cardStyles, styles];
```

**Template** — replace button class names:

```html
<button
  class="btn btn-blue"
  ?disabled=${this.classifyDisabled}
  @click=${this.handleClassify}
>
  Classify
</button>
<button class="btn btn-green" @click=${this.handleReview}>
  Review
</button>
<button
  class="btn btn-red"
  ?disabled=${this.classifying}
  @click=${this.handleDelete}
>
  Delete
</button>
```

**CSS removals** from `document-card.module.css`:

Remove the `.card` base block (shared `cardStyles` provides it):
```css
/* REMOVE */
.card {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  padding: var(--space-4);
  background: var(--bg-1);
  border: 1px solid var(--divider);
  border-radius: var(--radius-md);
  transition: border-color 0.15s;
}
```

Remove all button color blocks:
```css
/* REMOVE */
.classify-btn:not(:disabled) {
  border-color: var(--blue);
  color: var(--blue);

  &:hover {
    background: var(--blue-bg);
  }
}

.review-btn:not(:disabled) {
  border-color: var(--green);
  color: var(--green);

  &:hover {
    background: var(--green-bg);
  }
}

.delete-btn:not(:disabled) {
  border-color: var(--red);
  color: var(--red);

  &:hover {
    background: var(--red-bg);
  }
}
```

---

### `prompt-card.ts`

**Imports** — add `cardStyles`:

```typescript
import badgeStyles from "@styles/badge.module.css";
import buttonStyles from "@styles/buttons.module.css";
import cardStyles from "@styles/cards.module.css";
import styles from "./prompt-card.module.css";
```

**Static styles:**

```typescript
static styles = [buttonStyles, badgeStyles, cardStyles, styles];
```

**Template** — replace button class names:

```html
<button
  class="btn ${p.active ? "btn-yellow" : "btn-green"}"
  @click=${this.handleToggleActive}
>
  ${p.active ? "Deactivate" : "Activate"}
</button>
<button class="btn btn-red" @click=${this.handleDelete}>
  Delete
</button>
```

**CSS removals** from `prompt-card.module.css`:

Remove the `.card` base block (identical to document-card):
```css
/* REMOVE */
.card {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  padding: var(--space-4);
  background: var(--bg-1);
  border: 1px solid var(--divider);
  border-radius: var(--radius-md);
  transition: border-color 0.15s;
}
```

Remove all button color blocks:
```css
/* REMOVE */
.toggle-btn:not(:disabled) {
  border-color: var(--green);
  color: var(--green);

  &:hover {
    background: var(--green-bg);
  }
}

.toggle-btn.deactivate:not(:disabled) {
  border-color: var(--yellow);
  color: var(--yellow);

  &:hover {
    background: var(--yellow-bg);
  }
}

.delete-btn:not(:disabled) {
  border-color: var(--red);
  color: var(--red);

  &:hover {
    background: var(--red-bg);
  }
}
```

---

### `confirm-dialog.ts`

**Template** — replace button class name:

```html
<button class="btn btn-red" @click=${this.handleConfirm}>
  Confirm
</button>
```

**CSS removal** from `confirm-dialog.module.css`:

```css
/* REMOVE */
.confirm-btn:not(:disabled) {
  border-color: var(--red);
  color: var(--red);

  &:hover {
    background: var(--red-bg);
  }
}
```

---

### `document-grid.ts`

**Imports** — add `inputStyles`:

```typescript
import buttonStyles from "@styles/buttons.module.css";
import inputStyles from "@styles/inputs.module.css";
import styles from "./document-grid.module.css";
```

**Static styles:**

```typescript
static styles = [buttonStyles, inputStyles, styles];
```

**Template** — add `.input` class to search/filter/sort elements, replace button class:

```html
<div class="toolbar">
  <input
    type="search"
    class="input search-input"
    placeholder="Search documents..."
    .value=${this.search}
    @input=${this.handleSearchInput}
  />
  <select class="input filter-select" @change=${this.handleStatusFilter}>
    ...
  </select>
  <select class="input sort-select" @change=${this.handleSort}>
    ...
  </select>
  ${this.selectedIds.size > 0
    ? html`
        <button class="btn btn-blue" @click=${this.handleBulkClassify}>
          Classify ${this.selectedIds.size} Documents
        </button>
      `
    : nothing}
</div>
```

**CSS removals** from `document-grid.module.css`:

Remove the base input styling block (shared `inputStyles` provides it):
```css
/* REMOVE */
.search-input,
.filter-select,
.sort-select {
  padding: var(--space-2) var(--space-3);
  border: 1px solid var(--divider);
  border-radius: var(--radius-sm);
  background: var(--bg-1);
  color: var(--color);
  font-size: var(--text-sm);
  font-family: var(--font-mono);

  &:focus-visible {
    outline: 2px solid var(--blue);
    outline-offset: 2px;
  }
}
```

Remove the button color block:
```css
/* REMOVE */
.bulk-btn:not(:disabled) {
  border-color: var(--blue);
  color: var(--blue);

  &:hover {
    background: var(--blue-bg);
  }
}
```

**Retained** — layout-specific overrides stay in module CSS:
```css
/* KEEP */
.search-input {
  flex: 1;
  min-width: 12rem;
}
```

---

### `prompt-list.ts`

**Imports** — add `inputStyles`:

```typescript
import buttonStyles from "@styles/buttons.module.css";
import inputStyles from "@styles/inputs.module.css";
import styles from "./prompt-list.module.css";
```

**Static styles:**

```typescript
static styles = [buttonStyles, inputStyles, styles];
```

**Template** — add `.input` class to search/filter/sort elements, replace button class:

```html
<div class="toolbar">
  <input
    type="search"
    class="input search-input"
    placeholder="Search prompts..."
    .value=${this.search}
    @input=${this.handleSearchInput}
  />
  <button class="btn btn-blue" @click=${this.handleNew}>New</button>
  <select class="input filter-select" @change=${this.handleStageFilter}>
    ...
  </select>
  <select class="input sort-select" @change=${this.handleSort}>
    ...
  </select>
</div>
```

**CSS removals** from `prompt-list.module.css`:

Remove the base input styling block:
```css
/* REMOVE */
.search-input,
.filter-select,
.sort-select {
  padding: var(--space-2) var(--space-3);
  border: 1px solid var(--divider);
  border-radius: var(--radius-sm);
  background: var(--bg-1);
  color: var(--color);
  font-size: var(--text-sm);
  font-family: var(--font-mono);

  &:focus-visible {
    outline: 2px solid var(--blue);
    outline-offset: 2px;
  }
}
```

Remove the button color block:
```css
/* REMOVE */
.new-btn:not(:disabled) {
  border-color: var(--blue);
  color: var(--blue);

  &:hover {
    background: var(--blue-bg);
  }
}
```

**Retained** — layout-specific overrides stay in module CSS:
```css
/* KEEP */
.search-input {
  grid-column: span 3;
}

.filter-select,
.sort-select {
  grid-column: span 2;
}
```

---

### `classification-panel.ts`

**Imports** — add `inputStyles` and `labelStyles`:

```typescript
import badgeStyles from "@styles/badge.module.css";
import buttonStyles from "@styles/buttons.module.css";
import inputStyles from "@styles/inputs.module.css";
import labelStyles from "@styles/labels.module.css";
import styles from "./classification-panel.module.css";
```

**Static styles:**

```typescript
static styles = [badgeStyles, buttonStyles, inputStyles, labelStyles, styles];
```

**Template — `renderViewMode()`** — replace button classes, add `.label` to section labels:

```html
<div class="section">
  <span class="label">Classification</span>
  ...
</div>

<div class="section">
  <span class="label">Markings Found</span>
  ...
</div>

<div class="section">
  <span class="label">Rationale</span>
  ...
</div>

...

<div class="actions">
  <button
    class="btn btn-green"
    @click=${() => (this.mode = "validate")}
  >
    Validate
  </button>
  <button class="btn btn-blue" @click=${() => (this.mode = "update")}>
    Update
  </button>
</div>
```

**Template — `renderValidateMode()`** — add `.label` to labels, `.input` to inputs, replace button classes:

```html
<div class="field">
  <label class="label" for="validated_by">Your Name</label>
  <input
    class="input"
    id="validated_by"
    name="validated_by"
    type="text"
    required
    .value=${this.classification?.validated_by ?? ""}
  />
</div>

...

<div class="actions">
  <button
    type="submit"
    class="btn btn-green"
    ?disabled=${this.submitting}
  >
    ${this.submitting ? "Validating..." : "Validate"}
  </button>
  <button
    type="button"
    class="btn btn-muted"
    @click=${this.handleCancel}
    ?disabled=${this.submitting}
  >
    Cancel
  </button>
</div>
```

**Template — `renderUpdateMode()`** — add `.label` to labels, `.input` to inputs, replace button classes:

```html
<div class="field">
  <label class="label" for="classification">Classification</label>
  <input
    class="input"
    id="classification"
    name="classification"
    type="text"
    required
    .value=${c.classification}
  />
</div>

<div class="field">
  <label class="label" for="rationale">Rationale</label>
  <textarea
    class="input"
    id="rationale"
    name="rationale"
    required
    .value=${c.rationale}
  ></textarea>
</div>

<div class="field">
  <label class="label" for="updated_by">Your Name</label>
  <input class="input" id="updated_by" name="updated_by" type="text" required />
</div>

...

<div class="actions">
  <button
    type="submit"
    class="btn btn-blue"
    ?disabled=${this.submitting}
  >
    ${this.submitting ? "Updating..." : "Update"}
  </button>
  <button
    type="button"
    class="btn btn-muted"
    @click=${this.handleCancel}
    ?disabled=${this.submitting}
  >
    Cancel
  </button>
</div>
```

**CSS removals** from `classification-panel.module.css`:

Remove section label block:
```css
/* REMOVE */
.section-label {
  font-size: var(--text-xs);
  font-family: var(--font-mono);
  color: var(--color-1);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}
```

Remove field label block:
```css
/* REMOVE */
.field label {
  font-size: var(--text-xs);
  font-family: var(--font-mono);
  color: var(--color-1);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}
```

Remove field input/textarea block:
```css
/* REMOVE */
.field input,
.field textarea {
  padding: var(--space-2) var(--space-3);
  border: 1px solid var(--divider);
  border-radius: var(--radius-sm);
  background: var(--bg-1);
  color: var(--color);
  font-size: var(--text-sm);
  font-family: var(--font-mono);

  &:focus-visible {
    outline: 2px solid var(--blue);
    outline-offset: 2px;
  }

  &:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }
}
```

Remove all button color blocks:
```css
/* REMOVE */
.validate-btn:not(:disabled) {
  border-color: var(--green);
  color: var(--green);

  &:hover {
    background: var(--green-bg);
  }
}

.update-btn:not(:disabled) {
  border-color: var(--blue);
  color: var(--blue);

  &:hover {
    background: var(--blue-bg);
  }
}

.cancel-btn:not(:disabled) {
  border-color: var(--color-2);
  color: var(--color-2);

  &:hover {
    background: var(--bg-2);
  }
}
```

**Retained** — textarea sizing override stays in module CSS:
```css
/* KEEP */
.field textarea {
  resize: vertical;
  min-height: 6rem;
}
```

---

### `prompt-form.ts`

**Imports** — add `inputStyles` and `labelStyles`:

```typescript
import buttonStyles from "@styles/buttons.module.css";
import inputStyles from "@styles/inputs.module.css";
import labelStyles from "@styles/labels.module.css";
import styles from "./prompt-form.module.css";
```

**Static styles:**

```typescript
static styles = [buttonStyles, inputStyles, labelStyles, styles];
```

**Template** — add `.label` to labels, `.input` to inputs/selects/textareas, replace button classes:

```html
<div class="field">
  <label class="label" for="name">Name</label>
  <input
    class="input"
    id="name"
    name="name"
    type="text"
    required
    .value=${p?.name ?? ""}
  />
</div>
<div class="field">
  <label class="label" for="stage">Stage</label>
  <select
    class="input"
    id="stage"
    name="stage"
    required
    ?disabled=${this.isEdit}
    @change=${this.handleStageChange}
  >
    ...
  </select>
</div>
${this.renderDefaults()}
<div class="field">
  <label class="label" for="instructions">Instructions</label>
  <textarea
    class="input instructions"
    id="instructions"
    name="instructions"
    required
    .value=${p?.instructions ?? ""}
  ></textarea>
</div>
<div class="field">
  <label class="label" for="description">Description</label>
  <textarea
    class="input"
    id="descripion"
    name="description"
    .value=${p?.description ?? ""}
  ></textarea>
</div>
${this.renderError()}
<div class="actions">
  <button
    type="submit"
    class="btn btn-green"
    ?disabled=${this.submitting}
  >
    ${this.submitting ? "Saving..." : "Save"}
  </button>
  <button
    type="button"
    class="btn btn-muted"
    @click=${this.handleCancel}
    ?disabled=${this.submitting}
  >
    Cancel
  </button>
</div>
```

**`renderDefaults()`** — no changes. The `<h4>` headings inside the defaults panel are content headings, not form labels. They retain their own styling in component CSS.

**CSS removals** from `prompt-form.module.css`:

Remove field label block:
```css
/* REMOVE */
.field label {
  font-size: var(--text-xs);
  font-family: var(--font-mono);
  color: var(--color-1);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}
```

Remove field input/select/textarea block:
```css
/* REMOVE */
.field input,
.field select,
.field textarea {
  padding: var(--space-2) var(--space-3);
  border: 1px solid var(--divider);
  border-radius: var(--radius-sm);
  background: var(--bg-1);
  color: var(--color);
  font-size: var(--text-sm);
  font-family: var(--font-mono);

  &:focus-visible {
    outline: 2px solid var(--blue);
    outline-offset: 2px;
  }

  &:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }
}
```

**Retained** — defaults-content h4 stays as-is (content heading, not a form label):
```css
/* KEEP */
.defaults-content h4 {
  font-size: var(--text-xs);
  font-family: var(--font-mono);
  color: var(--color-1);
  text-transform: uppercase;
  letter-spacing: 0.05em;
  margin: 0;
}
```

Remove all button color blocks:
```css
/* REMOVE */
.save-btn:not(:disabled) {
  border-color: var(--green);
  color: var(--green);

  &:hover {
    background: var(--green-bg);
  }
}

.cancel-btn:not(:disabled) {
  border-color: var(--color-2);
  color: var(--color-2);

  &:hover {
    background: var(--bg-2);
  }
}
```

**Retained** — textarea sizing overrides stay in module CSS:
```css
/* KEEP */
.field textarea {
  resize: vertical;
  min-height: 6rem;
}

.field textarea.instructions {
  min-height: 12rem;
  flex: 1;
}
```

---

### `document-upload.ts`

**Template** — replace button class names:

```html
<button
  class="btn btn-yellow"
  @click=${this.handleClear}
  ?disabled=${this.uploading}
>
  ${allSettled ? "Done" : "Clear"}
</button>

<button
  class="btn btn-blue"
  @click=${this.handleUpload}
  ?disabled=${!this.canUpload}
>
  Upload
</button>

<button
  class="btn btn-red"
  @click=${() => this.handleRemove(index)}
>
  Remove
</button>
```

**CSS removals** from `document-upload.module.css`:

```css
/* REMOVE */
.upload-btn:not(:disabled) {
  border-color: var(--blue);
  color: var(--blue);

  &:hover {
    background: var(--blue-bg);
  }
}

.remove-btn {
  padding: var(--space-1) var(--space-2);
  border-color: var(--red);
  color: var(--red);

  &:hover:not(:disabled) {
    background: var(--red-bg);
  }
}

.clear-btn:not(:disabled) {
  border-color: var(--yellow);
  color: var(--yellow);

  &:hover {
    background: var(--yellow-bg);
  }
}
```

**Note**: The `.remove-btn` had `padding: var(--space-1) var(--space-2)` for compact sizing. After removal, add a component-specific padding override if the default `.btn` padding is too large. If it looks fine with the default, no override needed.

**Note**: `.field-input` in upload uses different sizing (`--space-1`, `--text-xs`, `--bg` not `--bg-1`) — keep component-specific, do not extract to shared `inputStyles`.

## 3. Classification Panel — Disable Actions When Validated

In `classification-panel.ts`, `renderViewMode()`:

```typescript
private get isValidated(): boolean {
  return !!this.classification?.validated_by;
}
```

Add `?disabled=${this.isValidated}` to both the Validate and Update buttons.

## 4. Document Card — Show External ID and Platform

In `document-card.ts`, add to the `.meta` section in `render()`:

```typescript
<span>${doc.external_platform} #${doc.external_id}</span>
```

Renders as e.g. "GitHub #12345" alongside existing page count, size, and date metadata.

## 5. Fix `querySelector<any>` Type Erasure

**`documents-view.ts:16`**: Change `querySelector<any>("hd-document-grid")` → `querySelector("hd-document-grid")`

**`prompts-view.ts:33`**: Change `querySelector<any>("hd-prompt-list")` → `querySelector("hd-prompt-list")`

The `HTMLElementTagNameMap` declarations on each component already provide the correct return type when using the tag name string literal.

## 6. Domain Barrel Exports — Named Exports Only

**`app/client/domains/prompts/index.ts`**:
```typescript
export type {
  Prompt,
  PromptStage,
  StageContent,
  CreatePromptCommand,
  UpdatePromptCommand,
  SearchRequest,
} from "./prompt";
export { PromptService } from "./service";
```

**`app/client/domains/storage/index.ts`**:
```typescript
export type { BlobMeta, BlobList } from "./blob";
export { StorageService } from "./service";
export type { StorageListParams } from "./service";
```

## Verification

1. `bun run build` succeeds
2. `go vet ./...` passes
3. Visual verification:
   - Documents view: cards render with external_id/platform, button colors match
   - Prompts view: form inputs, labels, button colors match
   - Review view: classification panel validate/update disabled when validated
   - All views: no visual regressions from CSS extraction
