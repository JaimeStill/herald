# 82 - Prompt Card Pure Element and Search Request Type

## Problem Context

First sub-issue of Objective #60 (Prompt Management View). The prompt list module (#83) and form/view (#84) need a pure display element for prompt cards and a typed search request for server-side filtering. The prompts domain layer (types + service) and backend API are already complete — this is purely client-side work.

## Architecture Approach

Follow established pure element pattern from `hd-document-card`: property-driven rendering, custom events for parent coordination, no service imports. Each domain owns its own `SearchRequest` type (matching the documents domain convention) to eventually retire the shared `PageRequest` interface from `@core`.

Stage badges reuse existing color tokens: blue (classify), orange (enhance), green (finalize).

Flatten `ui/elements/`, `ui/modules/`, and `ui/views/` — remove domain subdirectories. Components live directly in their tier directory. The `hd-` prefix already namespaces by domain, making subdirectories unnecessary ceremony.

## Implementation

### Step 0: Flatten UI directory structure

Remove domain subdirectories from `ui/elements/`, `ui/modules/`, and `ui/views/`. Move all `.ts` and `.module.css` files up one level, delete subdirectory `index.ts` barrels, and rewrite the tier-level barrels.

#### 0a. Flatten `ui/elements/`

Move files up to `ui/elements/`:

```
ui/elements/dialog/confirm-dialog.ts          → ui/elements/confirm-dialog.ts
ui/elements/dialog/confirm-dialog.module.css  → ui/elements/confirm-dialog.module.css
ui/elements/documents/classify-progress.ts          → ui/elements/classify-progress.ts
ui/elements/documents/classify-progress.module.css  → ui/elements/classify-progress.module.css
ui/elements/documents/document-card.ts              → ui/elements/document-card.ts
ui/elements/documents/document-card.module.css      → ui/elements/document-card.module.css
ui/elements/pagination/pagination-controls.ts          → ui/elements/pagination-controls.ts
ui/elements/pagination/pagination-controls.module.css  → ui/elements/pagination-controls.module.css
```

Delete the subdirectories (`dialog/`, `documents/`, `pagination/`) and their `index.ts` barrels.

Rewrite `ui/elements/index.ts`:

```ts
export { ClassifyProgress } from "./classify-progress";
export { ConfirmDialog } from "./confirm-dialog";
export { DocumentCard } from "./document-card";
export { PaginationControls } from "./pagination-controls";
```

#### 0b. Flatten `ui/modules/`

Move files up to `ui/modules/`:

```
ui/modules/documents/document-grid.ts          → ui/modules/document-grid.ts
ui/modules/documents/document-grid.module.css  → ui/modules/document-grid.module.css
ui/modules/documents/document-upload.ts          → ui/modules/document-upload.ts
ui/modules/documents/document-upload.module.css  → ui/modules/document-upload.module.css
```

Delete the `documents/` subdirectory and its `index.ts` barrel.

Rewrite `ui/modules/index.ts`:

```ts
export { DocumentGrid } from "./document-grid";
export { DocumentUpload } from "./document-upload";
```

#### 0c. Flatten `ui/views/`

Move files up to `ui/views/`:

```
ui/views/documents/documents-view.ts          → ui/views/documents-view.ts
ui/views/documents/documents-view.module.css  → ui/views/documents-view.module.css
ui/views/not-found/not-found-view.ts          → ui/views/not-found-view.ts
ui/views/not-found/not-found-view.module.css  → ui/views/not-found-view.module.css
ui/views/prompts/prompts-view.ts              → ui/views/prompts-view.ts
ui/views/prompts/prompts-view.module.css      → ui/views/prompts-view.module.css
ui/views/review/review-view.ts                → ui/views/review-view.ts
ui/views/review/review-view.module.css        → ui/views/review-view.module.css
```

Delete all subdirectories (`documents/`, `not-found/`, `prompts/`, `review/`) and their `index.ts` barrels.

Rewrite `ui/views/index.ts`:

```ts
export { DocumentsView } from "./documents-view";
export { NotFoundView } from "./not-found-view";
export { PromptsView } from "./prompts-view";
export { ReviewView } from "./review-view";
```

#### 0d. Invert router route dependency

Move route definitions out of the router package and into the app entry point level:

1. Update `Router` constructor to accept routes: `constructor(containerId: string, routes: Record<string, RouteConfig>)`. Store as `this.routes` and replace all `routes[...]` references with `this.routes[...]`. Remove `import { routes } from "./routes"`.

2. Move `app/client/core/router/routes.ts` → `app/client/routes.ts`. Update its import to use `@core/router` alias.

3. Update `app.ts` to import routes and pass to constructor: `new Router("app-content", routes)`.

#### 0e. No other import changes needed

All component files use `./` relative imports for their own CSS modules, which remain co-located after flattening. Cross-package imports use `@core`, `@domains/*`, `@styles/*` aliases — unchanged. `app.ts` imports `@ui/elements`, `@ui/modules`, `@ui/views` — unchanged (barrel paths are the same).

### Step 1: Add `SearchRequest` type to prompts domain

**File:** `app/client/domains/prompts/prompt.ts`

Add after the `UpdatePromptCommand` interface:

```ts
export interface SearchRequest {
  page?: number;
  page_size?: number;
  search?: string;
  sort?: string;
  stage?: PromptStage;
  name?: string;
  active?: boolean;
}
```

### Step 2: Migrate `PromptService` to `SearchRequest`

**File:** `app/client/domains/prompts/service.ts`

1. Replace the `@core` import line:

```ts
// Before
import { request, toQueryString } from "@core";
import type { PageRequest, PageResult, Result } from "@core";

// After
import { request, toQueryString } from "@core";
import type { PageResult, Result } from "@core";
```

2. Add `SearchRequest` to the local import:

```ts
// Before
import type {
  Prompt,
  PromptStage,
  StageContent,
  CreatePromptCommand,
  UpdatePromptCommand,
} from "./prompt";

// After
import type {
  Prompt,
  PromptStage,
  SearchRequest,
  StageContent,
  CreatePromptCommand,
  UpdatePromptCommand,
} from "./prompt";
```

3. Update method signatures:

```ts
// list() — change PageRequest to SearchRequest
async list(params?: SearchRequest): Promise<Result<PageResult<Prompt>>> {

// search() — change PageRequest to SearchRequest
async search(body: SearchRequest): Promise<Result<PageResult<Prompt>>> {
```

### Step 3: Add stage badge variants

**File:** `app/client/design/styles/badge.module.css`

Consolidate by color and add stage variants. Full replacement:

```css
.badge {
  flex-shrink: 0;
  padding: var(--space-1) var(--space-2);
  border-radius: var(--radius-sm);
  font-size: var(--text-xs);
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;

  &.pending {
    color: var(--yellow);
    background: var(--yellow-bg);
  }

  &.enhance {
    color: var(--orange);
    background: var(--orange-bg);
  }

  &.classify,
  &.finalize,
  &.review,
  &.uploading {
    color: var(--blue);
    background: var(--blue-bg);
  }

  &.success,
  &.complete {
    color: var(--green);
    background: var(--green-bg);
  }

  &.error {
    color: var(--red);
    background: var(--red-bg);
  }
}
```

### Step 4: Create prompt card CSS module

**New file:** `app/client/ui/elements/prompt-card.module.css`

```css
:host {
  display: block;
}

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

.card.selected {
  border-color: var(--blue);
}

.header {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  cursor: pointer;
}

.name {
  flex: 1;
  font-weight: 600;
  color: var(--color);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  min-width: 0;
}

.active-indicator {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
  background: var(--divider);
}

.active-indicator.active {
  background: var(--green);
}

.description {
  font-size: var(--text-sm);
  color: var(--color-2);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.actions {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

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

### Step 5: Create prompt card element

**New file:** `app/client/ui/elements/prompt-card.ts`

```ts
import { LitElement, html, nothing } from "lit";
import { customElement, property } from "lit/decorators.js";

import type { Prompt } from "@domains/prompts";

import badgeStyles from "@styles/badge.module.css";
import buttonStyles from "@styles/buttons.module.css";
import styles from "./prompt-card.module.css";

@customElement("hd-prompt-card")
export class PromptCard extends LitElement {
  static styles = [buttonStyles, badgeStyles, styles];

  @property({ type: Object }) prompt!: Prompt;
  @property({ type: Boolean }) selected = false;

  private handleSelect() {
    this.dispatchEvent(
      new CustomEvent("select", {
        detail: { id: this.prompt.id },
        bubbles: true,
        composed: true,
      }),
    );
  }

  private handleToggleActive() {
    this.dispatchEvent(
      new CustomEvent("toggle-active", {
        detail: { id: this.prompt.id, active: !this.prompt.active },
        bubbles: true,
        composed: true,
      }),
    );
  }

  private handleDelete() {
    this.dispatchEvent(
      new CustomEvent("delete", {
        detail: { prompt: this.prompt },
        bubbles: true,
        composed: true,
      }),
    );
  }

  private renderDescription() {
    if (!this.prompt.description) return nothing;

    return html`
      <div class="description">${this.prompt.description}</div>
    `;
  }

  render() {
    const p = this.prompt;

    return html`
      <div class="card ${this.selected ? "selected" : ""}">
        <div class="header" @click=${this.handleSelect}>
          <span class="name">${p.name}</span>
          <span class="badge ${p.stage}">${p.stage}</span>
          <span
            class="active-indicator ${p.active ? "active" : ""}"
            title=${p.active ? "Active" : "Inactive"}
          ></span>
        </div>

        ${this.renderDescription()}

        <div class="actions">
          <button
            class="btn toggle-btn ${p.active ? "deactivate" : ""}"
            @click=${this.handleToggleActive}
          >
            ${p.active ? "Deactivate" : "Activate"}
          </button>
          <button class="btn delete-btn" @click=${this.handleDelete}>
            Delete
          </button>
        </div>
      </div>
    `;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    "hd-prompt-card": PromptCard;
  }
}
```

### Step 6: Update elements barrel

**File:** `app/client/ui/elements/index.ts`

Add `PromptCard` export (file was already rewritten in Step 0a — just add the new line):

```ts
export { PromptCard } from "./prompt-card";
```

## Validation Criteria

- [ ] `ui/elements/`, `ui/modules/`, `ui/views/` flattened — no domain subdirectories
- [ ] All tier-level barrels rewritten with direct exports
- [ ] `SearchRequest` interface exported from prompts domain with `stage`, `name`, `active` fields
- [ ] `PromptService.list()` and `search()` accept `SearchRequest`
- [ ] `hd-prompt-card` renders prompt name, stage badge, active indicator, and truncated description
- [ ] Stage badge variants styled distinctly: classify (blue), enhance (orange), finalize (green)
- [ ] Card dispatches `select`, `toggle-active`, and `delete` custom events
- [ ] Card highlights when `selected` property is true
- [ ] Barrel exports updated with `PromptCard`
- [ ] `bun run build` succeeds with no errors
