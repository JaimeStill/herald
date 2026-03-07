# 89 - Markings List Element and Classification Panel Module

## Problem Context

Sub-issue 2 of Objective 5: Document Review View (#61). The review view needs a panel to display classification data alongside a PDF viewer. This issue creates the classification panel module (loads and displays classification, with validate/update actions) and a markings list element (renders security markings as badge tags).

## Architecture Approach

Two components following existing patterns:
- **`hd-markings-list`** — pure element like `prompt-card.ts`: receives data via `@property()`, renders badges, no events
- **`hd-classification-panel`** — stateful module like `prompt-form.ts`: loads data in `updated()`, manages mode state for view/validate/update, calls services, dispatches events

Confidence values (`HIGH`, `MEDIUM`, `LOW`) need badge color mapping. Since these don't match existing badge classes, we add three CSS classes in the panel's module CSS to keep badge styling self-contained.

## Implementation

### Step 1: `markings-list.module.css`

Create `app/client/ui/elements/markings-list.module.css`:

```css
:host {
  display: block;
}

.markings {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2);
}

.empty {
  font-size: var(--text-sm);
  color: var(--color-2);
  font-style: italic;
}
```

### Step 2: `markings-list.ts`

Create `app/client/ui/elements/markings-list.ts`:

```typescript
import { LitElement, html, nothing } from "lit";
import { customElement, property } from "lit/decorators.js";

import badgeStyles from "@styles/badge.module.css";
import styles from "./markings-list.module.css";

@customElement("hd-markings-list")
export class MarkingsList extends LitElement {
  static styles = [badgeStyles, styles];

  @property({ type: Array }) markings: string[] = [];

  render() {
    if (!this.markings.length) {
      return html`<span class="empty">No markings found</span>`;
    }

    return html`
      <div class="markings">
        ${this.markings.map(
          (m) => html`<span class="badge">${m}</span>`,
        )}
      </div>
    `;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    "hd-markings-list": MarkingsList;
  }
}
```

### Step 3: `classification-panel.module.css`

Create `app/client/ui/modules/classification-panel.module.css`:

```css
:host {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
  flex: 1;
  min-height: 0;
}

.panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-shrink: 0;
}

.panel-header h2 {
  font-size: var(--text-lg);
  font-weight: 600;
}

.panel-body {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
  padding-inline: var(--space-2);
  flex: 1;
  min-height: 0;
  overflow-y: auto;
}

.section {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}

.section-label {
  font-size: var(--text-xs);
  font-family: var(--font-mono);
  color: var(--color-1);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.classification-value {
  display: flex;
  align-items: center;
  gap: var(--space-3);
}

.classification-name {
  font-size: var(--text-lg);
  font-weight: 600;
  font-family: var(--font-mono);
}

.confidence.high {
  color: var(--green);
  background: var(--green-bg);
}

.confidence.medium {
  color: var(--yellow);
  background: var(--yellow-bg);
}

.confidence.low {
  color: var(--red);
  background: var(--red-bg);
}

.rationale {
  margin: 0;
  font-size: var(--text-sm);
  font-family: var(--font-mono);
  color: var(--color-2);
  white-space: pre-wrap;
}

.meta {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-3);
  font-size: var(--text-xs);
  color: var(--color-2);
}

.meta span {
  font-family: var(--font-mono);
}

.actions {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  flex-shrink: 0;
}

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

.field {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}

.field label {
  font-size: var(--text-xs);
  font-family: var(--font-mono);
  color: var(--color-1);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

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

.field textarea {
  resize: vertical;
  min-height: 6rem;
}

.error {
  font-size: var(--text-sm);
  font-family: var(--font-mono);
  color: var(--red);
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-4);
  color: var(--color-2);
  text-align: center;
}

.empty-state a {
  color: var(--blue);
  text-decoration: none;
  font-family: var(--font-mono);

  &:hover {
    text-decoration: underline;
  }
}

.loading {
  font-size: var(--text-sm);
  color: var(--color-2);
  font-style: italic;
}

.validated {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  font-size: var(--text-sm);
  color: var(--green);
  font-family: var(--font-mono);
}
```

### Step 4: `classification-panel.ts`

Create `app/client/ui/modules/classification-panel.ts`:

```typescript
import { LitElement, html, nothing } from "lit";
import { customElement, property, state } from "lit/decorators.js";

import { formatDate } from "@core/formatting";
import { navigate } from "@core/router";
import { ClassificationService } from "@domains/classifications";
import type { Classification } from "@domains/classifications";

import badgeStyles from "@styles/badge.module.css";
import buttonStyles from "@styles/buttons.module.css";
import styles from "./classification-panel.module.css";

type PanelMode = "view" | "validate" | "update";

@customElement("hd-classification-panel")
export class ClassificationPanel extends LitElement {
  static styles = [buttonStyles, badgeStyles, styles];

  @property() documentId = "";

  @state() private classification: Classification | null = null;
  @state() private loading = true;
  @state() private error = "";
  @state() private mode: PanelMode = "view";
  @state() private submitting = false;

  updated(changed: Map<string, unknown>) {
    if (changed.has("documentId") && this.documentId) {
      this.loadClassification();
    }
  }

  private async loadClassification() {
    this.loading = true;
    this.error = "";
    this.classification = null;

    const result = await ClassificationService.findByDocument(this.documentId);

    this.loading = false;

    if (!result.ok) {
      this.error = result.error;
      return;
    }

    this.classification = result.data;
  }

  private async handleValidate(e: SubmitEvent) {
    e.preventDefault();
    this.submitting = true;

    const form = e.target as HTMLFormElement;
    const data = new FormData(form);
    const validated_by = (data.get("validated_by") as string).trim();

    const result = await ClassificationService.validate(
      this.classification!.id,
      { validated_by },
    );

    this.submitting = false;

    if (!result.ok) {
      this.error = result.error;
      return;
    }

    this.classification = result.data;
    this.mode = "view";
    this.error = "";

    this.dispatchEvent(
      new CustomEvent("validate", {
        detail: { classification: result.data },
        bubbles: true,
        composed: true,
      }),
    );
  }

  private async handleUpdate(e: SubmitEvent) {
    e.preventDefault();
    this.submitting = true;

    const form = e.target as HTMLFormElement;
    const data = new FormData(form);

    const command = {
      classification: (data.get("classification") as string).trim(),
      rationale: (data.get("rationale") as string).trim(),
      updated_by: (data.get("updated_by") as string).trim(),
    };

    const result = await ClassificationService.update(
      this.classification!.id,
      command,
    );

    this.submitting = false;

    if (!result.ok) {
      this.error = result.error;
      return;
    }

    this.classification = result.data;
    this.mode = "view";
    this.error = "";

    this.dispatchEvent(
      new CustomEvent("update", {
        detail: { classification: result.data },
        bubbles: true,
        composed: true,
      }),
    );
  }

  private handleBack() {
    navigate("");
  }

  private handleCancel() {
    this.mode = "view";
    this.error = "";
  }

  private renderError() {
    if (!this.error) return nothing;
    return html`<div class="error">${this.error}</div>`;
  }

  private renderValidated() {
    const c = this.classification;
    if (!c?.validated_by) return nothing;

    return html`
      <div class="validated">
        <span>Validated by ${c.validated_by}</span>
        ${c.validated_at ? html`<span>on ${formatDate(c.validated_at)}</span>` : nothing}
      </div>
    `;
  }

  private renderViewMode() {
    const c = this.classification!;

    return html`
      <div class="panel-body">
        <div class="section">
          <span class="section-label">Classification</span>
          <div class="classification-value">
            <span class="classification-name">${c.classification}</span>
            <span class="badge confidence ${c.confidence.toLowerCase()}">${c.confidence}</span>
          </div>
        </div>

        <div class="section">
          <span class="section-label">Markings Found</span>
          <hd-markings-list .markings=${c.markings_found}></hd-markings-list>
        </div>

        <div class="section">
          <span class="section-label">Rationale</span>
          <pre class="rationale">${c.rationale}</pre>
        </div>

        ${this.renderValidated()}

        <div class="meta">
          <span>${c.model_name} / ${c.provider_name}</span>
          <span>Classified ${formatDate(c.classified_at)}</span>
        </div>
      </div>

      <div class="actions">
        <button
          class="btn validate-btn"
          @click=${() => (this.mode = "validate")}
        >
          Validate
        </button>
        <button
          class="btn update-btn"
          @click=${() => (this.mode = "update")}
        >
          Update
        </button>
      </div>
    `;
  }

  private renderValidateMode() {
    return html`
      <form class="panel-body" @submit=${this.handleValidate}>
        <p>Confirm that the AI classification is correct.</p>

        <div class="field">
          <label for="validated_by">Your Name</label>
          <input
            id="validated_by"
            name="validated_by"
            type="text"
            required
            .value=${this.classification?.validated_by ?? ""}
          />
        </div>

        ${this.renderError()}

        <div class="actions">
          <button
            type="submit"
            class="btn validate-btn"
            ?disabled=${this.submitting}
          >
            ${this.submitting ? "Validating..." : "Validate"}
          </button>
          <button
            type="button"
            class="btn cancel-btn"
            @click=${this.handleCancel}
            ?disabled=${this.submitting}
          >
            Cancel
          </button>
        </div>
      </form>
    `;
  }

  private renderUpdateMode() {
    const c = this.classification!;

    return html`
      <form class="panel-body" @submit=${this.handleUpdate}>
        <p>Manually update the classification result.</p>

        <div class="field">
          <label for="classification">Classification</label>
          <input
            id="classification"
            name="classification"
            type="text"
            required
            .value=${c.classification}
          />
        </div>

        <div class="field">
          <label for="rationale">Rationale</label>
          <textarea
            id="rationale"
            name="rationale"
            required
            .value=${c.rationale}
          ></textarea>
        </div>

        <div class="field">
          <label for="updated_by">Your Name</label>
          <input
            id="updated_by"
            name="updated_by"
            type="text"
            required
          />
        </div>

        ${this.renderError()}

        <div class="actions">
          <button
            type="submit"
            class="btn update-btn"
            ?disabled=${this.submitting}
          >
            ${this.submitting ? "Updating..." : "Update"}
          </button>
          <button
            type="button"
            class="btn cancel-btn"
            @click=${this.handleCancel}
            ?disabled=${this.submitting}
          >
            Cancel
          </button>
        </div>
      </form>
    `;
  }

  render() {
    if (this.loading) {
      return html`<span class="loading">Loading classification...</span>`;
    }

    if (!this.classification) {
      return html`
        <div class="empty-state">
          <span>No classification found for this document.</span>
          <button class="button" @click=${this.handleBack}>
            Back to Documents
          </button>
        </div>
      `;
    }

    return html`
      <div class="panel-header">
        <h2>Classification</h2>
      </div>

      ${this.mode === "view"
        ? this.renderViewMode()
        : this.mode === "validate"
          ? this.renderValidateMode()
          : this.renderUpdateMode()}
    `;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    "hd-classification-panel": ClassificationPanel;
  }
}
```

### Step 5: Barrel Updates

**`app/client/ui/elements/index.ts`** — add:

```typescript
export { MarkingsList } from "./markings-list";
```

**`app/client/ui/modules/index.ts`** — add:

```typescript
export { ClassificationPanel } from "./classification-panel";
```

## Remediation

### R1: Integrate classification panel into review view

The classification panel cannot be verified in isolation — it needs the review view to host it. The view already has two-panel flex layout with a placeholder right panel. Replace the placeholder with `<hd-classification-panel>` and wire up `validate`/`update` event listeners to re-fetch the document (for status changes). Remove the now-unused placeholder CSS rules.

> **Note:** This completes the review view composition originally scoped for Task #90. That task will be repurposed as a comprehensive evaluation of the overall web application architecture, layout, and functionality.

**`app/client/ui/views/review-view.ts`** — replace placeholder content with classification panel.

`app.ts` already imports `@ui/elements` and `@ui/modules` which registers all custom elements globally, so no component-level barrel imports are needed. Remove the redundant imports:

- `app/client/ui/modules/classification-panel.ts` — remove `import "@ui/elements";`
- `app/client/ui/views/review-view.ts` — remove `import "@ui/modules";` (if present from #88)

Add a `handleClassificationChange` method that re-fetches the document to reflect status transitions:

```typescript
private async handleClassificationChange() {
  if (!this.documentId) return;

  const result = await DocumentService.find(this.documentId);

  if (result.ok) {
    this.document = result.data;
  }
}
```

Replace the right panel content in `render()`:

```typescript
<div class="panel classification-panel">
  <hd-classification-panel
    .documentId=${this.documentId ?? ""}
    @validate=${this.handleClassificationChange}
    @update=${this.handleClassificationChange}
  ></hd-classification-panel>
</div>
```

**`app/client/ui/views/review-view.module.css`** — remove the placeholder rules that the panel module now owns:

Remove these rules (the `.classification-panel` container rule stays):

```css
.classification-panel h2 {
  margin-bottom: var(--space-2);
  font-family: var(--font-mono);
  font-size: var(--text-base);
  word-break: break-all;
}

.classification-panel .status {
  color: var(--color-1);
  font-family: var(--font-mono);
  font-size: var(--text-sm);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}
```

## Validation Criteria

- [ ] `bun run build` succeeds from `app/`
- [ ] `hd-markings-list` renders string array as styled badge tags
- [ ] `hd-markings-list` shows empty state when no markings
- [ ] `hd-classification-panel` loads and displays classification data for a document
- [ ] Validate action calls API and dispatches `validate` event with updated classification
- [ ] Update action expands form, calls API, dispatches `update` event with updated classification
- [ ] Empty state shown when no classification exists
- [ ] Review view renders classification panel in right panel
- [ ] Validate/update events trigger document re-fetch in review view
