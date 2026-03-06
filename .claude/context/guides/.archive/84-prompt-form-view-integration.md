# 84 - Prompt Form Module and View Integration

## Problem Context

Third and final sub-issue of Objective #60 (Prompt Management View). The prompt card (#82) and prompt list (#83) modules are merged. This task creates the `hd-prompt-form` module for create/edit operations and replaces the stub `hd-prompts-view` with a full split-panel layout composing the list and form.

## Architecture Approach

Follows the established view composition pattern from `hd-documents-view`:
- View manages UI state (`showForm`, `selectedPrompt`) with `@state()`
- Coordinates between list and form modules via custom events and `querySelector`
- Form uses `FormData` API for value extraction on submit (not controlled inputs)
- Stage dropdown disabled in edit mode to prevent accidental stage changes

## Implementation

### Step 1: Create `app/client/ui/modules/prompt-form.module.css`

New file — complete implementation:

```css
:host {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.form-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-shrink: 0;
}

.form-header h2 {
  font-size: var(--text-lg);
  font-weight: 600;
}

.form-body {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
  flex: 1;
  min-height: 0;
  overflow-y: auto;
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

.field textarea {
  resize: vertical;
  min-height: 6rem;
}

.field textarea.instructions {
  min-height: 12rem;
  flex: 1;
}

.actions {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  flex-shrink: 0;
}

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

.error {
  font-size: var(--text-sm);
  font-family: var(--font-mono);
  color: var(--red);
}
```

### Step 2: Create `app/client/ui/modules/prompt-form.ts`

New file — complete implementation:

```typescript
import { LitElement, html, nothing } from "lit";
import { customElement, property, state } from "lit/decorators.js";

import { PromptService } from "@domains/prompts";
import type { Prompt } from "@domains/prompts";

import buttonStyles from "@styles/buttons.module.css";
import styles from "./prompt-form.module.css";

@customElement("hd-prompt-form")
export class PromptForm extends LitElement {
  static styles = [buttonStyles, styles];

  @property({ type: Object }) prompt: Prompt | null = null;

  @state() private submitting = false;
  @state() private error = "";

  private get isEdit() {
    return this.prompt !== null;
  }

  private async handleSubmit(e: SubmitEvent) {
    e.preventDefault();
    this.error = "";
    this.submitting = true;

    const form = e.target as HTMLFormElement;
    const data = new FormData(form);

    const name = (data.get("name") as string).trim();
    const stage = data.get("stage") as string;
    const instructions = (data.get("instructions") as string).trim();
    const description = (data.get("description") as string).trim();

    const command = {
      name,
      stage: stage as Prompt["stage"],
      instructions,
      ...(description && { description }),
    };

    const result = this.isEdit
      ? await PromptService.update(this.prompt!.id, command)
      : await PromptService.create(command);

    this.submitting = false;

    if (!result.ok) {
      this.error = result.error ?? "An unexpected error occurred.";
      return;
    }

    this.dispatchEvent(
      new CustomEvent("save", {
        detail: { prompt: result.data },
        bubbles: true,
        composed: true,
      }),
    );
  }

  private handleCancel() {
    this.error = "";
    this.dispatchEvent(
      new CustomEvent("cancel", {
        bubbles: true,
        composed: true,
      }),
    );
  }

  private renderError() {
    if (!this.error) return nothing;
    return html`<div class="error">${this.error}</div>`;
  }

  render() {
    const p = this.prompt;

    return html`
      <div class="form-header">
        <h2>${this.isEdit ? "Edit Prompt" : "New Prompt"}</h2>
      </div>
      <form class="form-body" @submit=${this.handleSubmit}>
        <div class="field">
          <label for="name">Name</label>
          <input
            id="name"
            name="name"
            type="text"
            required
            .value=${p?.name ?? ""}
          />
        </div>
        <div class="field">
          <label for="stage">Stage</label>
          <select
            id="stage"
            name="stage"
            required
            ?disabled=${this.isEdit}
          >
            <option value="" ?selected=${!p}>---</option>
            <option value="classify" ?selected=${p?.stage === "classify"}>
              Classify
            </option>
            <option value="enhance" ?selected=${p?.stage === "enhance"}>
              Enhance
            </option>
            <option value="finalize" ?selected=${p?.stage === "finalize"}>
              Finalize
            </option>
          </select>
        </div>
        <div class="field">
          <label for="instructions">Instructions</label>
          <textarea
            id="instructions"
            name="instructions"
            class="instructions"
            required
            .value=${p?.instructions ?? ""}
          ></textarea>
        </div>
        <div class="field">
          <label for="description">Description</label>
          <textarea
            id="description"
            name="description"
            .value=${p?.description ?? ""}
          ></textarea>
        </div>
        ${this.renderError()}
        <div class="actions">
          <button
            type="submit"
            class="btn save-btn"
            ?disabled=${this.submitting}
          >
            ${this.submitting ? "Saving..." : "Save"}
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
}

declare global {
  interface HTMLElementTagNameMap {
    "hd-prompt-form": PromptForm;
  }
}
```

### Step 3: Update `app/client/ui/modules/index.ts`

Add the prompt form export:

```typescript
export { PromptForm } from "./prompt-form";
```

### Step 4: Replace `app/client/ui/views/prompts-view.module.css`

Replace entire file:

```css
:host {
  display: flex;
  flex-direction: column;
  padding: var(--space-4) var(--space-6);
  overflow: hidden;
}

.view {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
  flex: 1;
  min-height: 0;
}

.view-header {
  display: flex;
  align-items: center;
  flex-shrink: 0;
}

h1 {
  font-size: var(--text-xl);
  font-weight: 600;
}

.view-content {
  display: flex;
  gap: var(--space-6);
  flex: 1;
  min-height: 0;
}

.list-panel {
  display: flex;
  flex-direction: column;
  width: 22rem;
  min-width: 18rem;
  flex-shrink: 0;
  border-right: 1px solid var(--divider);
  padding-right: var(--space-6);
  overflow: hidden;
}

.form-panel {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-width: 0;
  overflow: hidden;
}
```

### Step 5: Replace `app/client/ui/views/prompts-view.ts`

Replace entire file:

```typescript
import { LitElement, html, nothing } from "lit";
import { customElement, state } from "lit/decorators.js";

import type { Prompt } from "@domains/prompts";

import styles from "./prompts-view.module.css";

@customElement("hd-prompts-view")
export class PromptsView extends LitElement {
  static styles = styles;

  @state() private selectedPrompt: Prompt | null = null;
  @state() private showForm = false;

  private handleCreate() {
    this.selectedPrompt = null;
    this.showForm = true;
  }

  private handlePromptSelect(e: CustomEvent<{ prompt: Prompt }>) {
    this.selectedPrompt = e.detail.prompt;
    this.showForm = true;
  }

  private handlePromptSaved() {
    this.showForm = false;
    this.selectedPrompt = null;
    this.renderRoot.querySelector<any>("hd-prompt-list")?.refresh();
  }

  private handleCancel() {
    this.showForm = false;
    this.selectedPrompt = null;
  }

  private handlePromptDeleted(e: CustomEvent<{ id: string }>) {
    if (this.selectedPrompt?.id === e.detail.id) {
      this.showForm = false;
      this.selectedPrompt = null;
    }
  }

  render() {
    return html`
      <div class="view">
        <div class="view-header">
          <h1>Prompts</h1>
        </div>
        <div class="view-content">
          <div class="list-panel">
            <hd-prompt-list
              .selected=${this.selectedPrompt}
              @create=${this.handleCreate}
              @select=${this.handlePromptSelect}
              @delete=${this.handlePromptDeleted}
            ></hd-prompt-list>
          </div>
          ${this.showForm
            ? html`
                <div class="form-panel">
                  <hd-prompt-form
                    .prompt=${this.selectedPrompt}
                    @save=${this.handlePromptSaved}
                    @cancel=${this.handleCancel}
                  ></hd-prompt-form>
                </div>
              `
            : nothing}
        </div>
      </div>
    `;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    "hd-prompts-view": PromptsView;
  }
}
```

## Remediation

### R1: Default prompt reference panel

Prompts have two layers: an immutable **specification** (return format, behavioral constraints) and transient **instructions** (optimizable per-prompt). Users need full context into the default behavior of the stage they're modifying.

This remediation adds three things:
1. A form description beneath the header explaining the relationship between specification and instructions
2. A `?default=true` query param on the Go instructions endpoint to bypass DB overrides
3. A collapsible `<details>` panel showing both the spec and default instructions for the selected stage

#### R1a: Go — add `default` query param to instructions endpoint

**`internal/prompts/handler.go`** — pass a `defaultOnly` flag to the system call:

In the `Instructions` handler method, read a `default` query param from the request. When `"true"`, call the hardcoded `Instructions(stage)` function directly instead of going through the repository (which checks for active DB overrides).

```go
func (h *Handler) Instructions(w http.ResponseWriter, r *http.Request) {
	stage, err := ParseStage(r.PathValue("stage"))
	if err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, err)
		return
	}

	var text string
	if r.URL.Query().Get("default") == "true" {
		text, err = Instructions(stage)
	} else {
		text, err = h.sys.Instructions(r.Context(), stage)
	}

	if err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(err), err)
		return
	}

	handlers.RespondJSON(w, http.StatusOK, StageContent{Stage: stage, Content: text})
}
```

This is backwards-compatible — existing calls without the param behave identically.

#### R1b: Client — add `default` param to service

**`app/client/domains/prompts/service.ts`** — add optional `defaultOnly` param to the `instructions` method:

```typescript
async instructions(
  stage: PromptStage,
  defaultOnly?: boolean,
): Promise<Result<StageContent>> {
  const params = defaultOnly ? "?default=true" : "";
  return await request<StageContent>(`${base}/${stage}/instructions${params}`);
},
```

#### R1c: Form — default prompt reference panel

**`app/client/ui/modules/prompt-form.ts`** — add state, fetch logic, description, and reference panel:

Rename the existing `@state() private instructions` to `@state() private defaultInstructions` (the existing `instructions` state conflicts with the form field name — the default instructions state tracks the hardcoded default fetched from the API, not the form textarea value).

Add `fetchDefaults` method — fetches both spec and default instructions in parallel:

```typescript
private async fetchDefaults(stage: string) {
  this.spec = "";
  this.defaultInstructions = "";

  const typedStage = stage as Prompt["stage"];

  const [specResult, instrResult] = await Promise.all([
    PromptService.spec(typedStage),
    PromptService.instructions(typedStage, true),
  ]);

  if (specResult.ok) this.spec = specResult.data.content;
  if (instrResult.ok) this.defaultInstructions = instrResult.data.content;
}
```

Add `updated()` lifecycle for edit mode — fetches defaults when the prompt property changes:

```typescript
updated(changed: Map<string, unknown>) {
  if (changed.has("prompt") && this.prompt) {
    this.fetchDefaults(this.prompt.stage);
  }
}
```

Add `handleStageChange` for create mode — wire to `@change` on the stage `<select>`:

```typescript
private handleStageChange(e: Event) {
  const stage = (e.target as HTMLSelectElement).value;

  if (stage) this.fetchDefaults(stage);
}
```

In the template, add `@change=${this.handleStageChange}` to the stage `<select>` element.

Add `renderDefaults()` method:

```typescript
private renderDefaults() {
  if (!this.spec && !this.defaultInstructions) return nothing;

  const stage = this.prompt?.stage
    ?? (this.renderRoot.querySelector<HTMLSelectElement>("#stage")?.value || "");

  const label = stage.charAt(0).toUpperCase() + stage.slice(1);

  return html`
    <details class="defaults">
      <summary>${label} — Default Prompt</summary>
      <div class="defaults-content">
        ${this.spec
          ? html`
              <h4>Specification</h4>
              <pre>${this.spec}</pre>
            `
          : nothing}
        ${this.defaultInstructions
          ? html`
              <h4>Default Instructions</h4>
              <pre>${this.defaultInstructions}</pre>
            `
          : nothing}
      </div>
    </details>
  `;
}
```

- Insert `${this.renderDefaults()}` in `render()` between the stage field and the instructions field
- Add a form description beneath the header:

```typescript
<div class="form-header">
  <h2>${this.isEdit ? "Edit Prompt" : "New Prompt"}</h2>
</div>
<p class="form-description">
  Instructions are combined with the stage specification to form the complete
  prompt. Override the default instructions below to customize behavior.
</p>
```

**`app/client/ui/modules/prompt-form.module.css`** — add styles:

```css
.form-description {
  font-size: var(--text-sm);
  color: var(--color-2);
  margin: 0;
}

.defaults {
  border: 1px solid var(--divider);
  border-radius: var(--radius-sm);
  font-size: var(--text-sm);
  background: var(--bg-1);
}

.defaults summary {
  padding: var(--space-2) var(--space-3);
  font-family: var(--font-mono);
  color: var(--color-1);
  cursor: pointer;
}

.defaults-content {
  padding: var(--space-3);
  border-top: 1px solid var(--divider);
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
  max-height: 20rem;
  overflow-y: auto;
}

.defaults-content h4 {
  font-size: var(--text-xs);
  font-family: var(--font-mono);
  color: var(--color-1);
  text-transform: uppercase;
  letter-spacing: 0.05em;
  margin: 0;
}

.defaults-content pre {
  margin: 0;
  font-family: var(--font-mono);
  font-size: var(--text-xs);
  color: var(--color-2);
  white-space: pre-wrap;
}
```

## Validation Criteria

- [ ] `bun run build` completes with no errors
- [ ] Form creates new prompts (name, stage, instructions, description)
- [ ] Form edits existing prompts (pre-populated fields, stage disabled)
- [ ] Form dispatches `prompt-saved` on success, `cancel` on cancel
- [ ] Form shows error messages on API failure
- [ ] View composes list and form in split layout
- [ ] Selecting a prompt populates the form in edit mode
- [ ] "New" button opens form in create mode
- [ ] Saving refreshes the list
- [ ] Deleting a selected prompt clears the form
- [ ] Instructions textarea uses monospace font
- [ ] `GET /api/prompts/{stage}/instructions?default=true` returns hardcoded default (bypasses DB)
- [ ] `GET /api/prompts/{stage}/instructions` (no param) behaves unchanged
- [ ] Default prompt details panel appears between stage and instructions when stage is known
- [ ] Panel is collapsed by default, shows both specification and default instructions
- [ ] Panel loads on stage dropdown change (create mode)
- [ ] Panel loads when prompt is set (edit mode)
- [ ] Form description text appears beneath the form header
