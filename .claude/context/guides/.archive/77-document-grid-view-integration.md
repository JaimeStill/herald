# 77 - Document Grid, View Integration, and Bulk Classify

## Problem Context

Final sub-issue of Objective #59 (Document Management View). Assembles the full document management interface by composing the service layer (#74), card + progress elements (#75), and upload component (#76). Also incorporates frontend design quality improvements (accessibility, modern CSS, shared styles) as this is the first view seen fully wired together.

## Architecture Approach

- **Data fetching**: Grid uses `POST /api/documents/search` via `DocumentService.search()` with a flat `SearchRequest` type matching Go's embedded struct JSON serialization
- **No external signals**: Grid manages its own state via `@state()` — only consumer of the document list. View refreshes grid via `querySelector` + method call
- **Upload toggle**: Hidden behind a button in the view header
- **Selection**: Grid owns `selectedIds` set, card receives `selected` boolean prop, dispatches `select` event
- **Pagination**: Extracted as reusable `hd-pagination` pure element for cross-view reuse

## Implementation

### Step 1: SearchRequest Convention and toQueryString

Each domain defines its own `SearchRequest` interface — pagination fields plus domain-specific filters. Both `list()` (GET + query params) and `search()` (POST + JSON body) accept it. `toQueryString` is generalized to serialize any flat object.

**`app/client/core/api.ts`** — Generalize `toQueryString` signature:

```typescript
/** Converts a flat params object to a query string (e.g., `?page=1&status=pending`). */
export function toQueryString<T extends object>(params: T): string {
  const entries = Object.entries(params)
    .filter(([, v]) => v !== undefined && v !== null && v !== '')
    .map(([k, v]) => `${encodeURIComponent(k)}=${encodeURIComponent(String(v))}`);

  return entries.length > 0
    ? `?${entries.join('&')}`
    : '';
}
```

Remove the `PageRequest` import dependency from `toQueryString` (keep it for other re-exports).

**`app/client/documents/document.ts`** — Add after the `Document` interface:

```typescript
/** Pagination + filter params for document list and search endpoints. */
export interface SearchRequest {
  page?: number;
  page_size?: number;
  search?: string;
  sort?: string;
  status?: string;
  classification?: string;
  confidence?: string;
}
```

**`app/client/documents/service.ts`** — Update both `list()` and `search()` to accept `SearchRequest`:

```typescript
import type { Document, SearchRequest } from './document';

// list() now accepts SearchRequest — filters go as query params
async list(params?: SearchRequest): Promise<Result<PageResult<Document>>> {
  return await request<PageResult<Document>>(
    `${base}${params ? toQueryString(params) : ''}`
  );
},

// search() also accepts SearchRequest — same type, POST body
async search(body: SearchRequest): Promise<Result<PageResult<Document>>> {
  return await request<PageResult<Document>>(`${base}/search`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  });
},
```

**`app/client/documents/index.ts`** — Add `SearchRequest` to exports:

```typescript
export type { Document, DocumentStatus, SearchRequest } from './document';
```

The same convention applies to classifications and prompts — each domain will define its own `SearchRequest` when those views are built. For now, only documents gets the type since it's the active view.

### Step 2: Design System Updates

**`app/client/design/core/tokens.css`** — Replace entire file with `light-dark()` consolidation:

```css
@layer tokens {
  :root {
    color-scheme: dark light;

    --font-sans: system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Oxygen, Ubuntu, Cantarell, sans-serif;
    --font-mono: ui-monospace, "Cascadia Code", "Source Code Pro", Menlo, Consolas, "DejaVu Sans Mono", monospace;

    --space-1: 0.25rem;
    --space-2: 0.5rem;
    --space-3: 0.75rem;
    --space-4: 1rem;
    --space-5: 1.25rem;
    --space-6: 1.5rem;
    --space-8: 2rem;
    --space-10: 2.5rem;
    --space-12: 3rem;
    --space-16: 4rem;

    --text-xs: 0.75rem;
    --text-sm: 0.875rem;
    --text-base: 1rem;
    --text-lg: 1.125rem;
    --text-xl: 1.25rem;
    --text-2xl: 1.5rem;
    --text-3xl: 1.875rem;
    --text-4xl: 2.25rem;

    --radius-sm: 0.25rem;
    --radius-md: 0.5rem;
    --radius-lg: 0.75rem;

    --shadow-sm: 0 1px 2px hsl(0 0% 0% / 0.05);
    --shadow-md: 0 4px 6px hsl(0 0% 0% / 0.1);
    --shadow-lg: 0 10px 15px hsl(0 0% 0% / 0.15);

    --bg: light-dark(hsl(0, 0%, 100%), hsl(0, 0%, 7%));
    --bg-1: light-dark(hsl(0, 0%, 96%), hsl(0, 0%, 12%));
    --bg-2: light-dark(hsl(0, 0%, 92%), hsl(0, 0%, 18%));
    --color: light-dark(hsl(0, 0%, 10%), hsl(0, 0%, 93%));
    --color-1: light-dark(hsl(0, 0%, 30%), hsl(0, 0%, 80%));
    --color-2: light-dark(hsl(0, 0%, 45%), hsl(0, 0%, 65%));
    --divider: light-dark(hsl(0, 0%, 80%), hsl(0, 0%, 25%));

    --blue: light-dark(hsl(210, 90%, 45%), hsl(210, 100%, 70%));
    --blue-bg: light-dark(hsl(210, 80%, 92%), hsl(210, 50%, 20%));
    --green: light-dark(hsl(140, 60%, 35%), hsl(140, 70%, 55%));
    --green-bg: light-dark(hsl(140, 50%, 90%), hsl(140, 40%, 18%));
    --red: light-dark(hsl(0, 70%, 50%), hsl(0, 85%, 65%));
    --red-bg: light-dark(hsl(0, 70%, 93%), hsl(0, 50%, 20%));
    --yellow: light-dark(hsl(45, 80%, 40%), hsl(45, 90%, 60%));
    --yellow-bg: light-dark(hsl(45, 80%, 88%), hsl(45, 50%, 18%));
    --orange: light-dark(hsl(25, 85%, 50%), hsl(25, 95%, 65%));
    --orange-bg: light-dark(hsl(25, 75%, 90%), hsl(25, 50%, 20%));
  }
}
```

**`app/client/design/index.css`** — Update layer declaration:

```css
@layer tokens, reset, theme, app;

@import url(./core/tokens.css);
@import url(./core/reset.css);
@import url(./core/theme.css);

@import url(./app/app.css);
```

**`app/client/design/app/app.css`** — Wrap in `@layer app`:

```css
@layer app {
  body {
    display: flex;
    flex-direction: column;
    height: 100dvh;
    margin: 0;
    overflow: hidden;
  }

  .app-header {
    flex-shrink: 0;
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: var(--space-3) var(--space-6);
    background: var(--bg-1);
    border-bottom: 1px solid var(--divider);
  }

  .app-header .brand {
    font-size: var(--text-lg);
    font-weight: 600;
    color: var(--color);
    text-decoration: none;

    &:hover {
      color: var(--blue);
    }
  }

  .app-header nav {
    display: flex;
    gap: var(--space-4);

    & a {
      color: var(--color-1);
      text-decoration: none;
      font-size: var(--text-sm);

      &:hover {
        color: var(--blue);
      }
    }
  }

  #app-content {
    flex: 1;
    display: flex;
    flex-direction: column;
    min-height: 0;
    overflow: hidden;

    & > * {
      flex: 1;
      min-height: 0;
    }
  }
}
```

**Delete `app/client/design/app/elements.css`** — Empty file.

**New file: `app/client/design/shared/buttons.module.css`**:

```css
.btn {
  padding: var(--space-1) var(--space-3);
  border: 1px solid var(--divider);
  border-radius: var(--radius-sm);
  background: var(--bg-2);
  color: var(--color);
  font-size: var(--text-xs);
  font-family: var(--font-sans);
  cursor: pointer;
  transition: background 0.15s, border-color 0.15s;

  &:hover:not(:disabled) {
    border-color: var(--color-2);
  }

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

**New file: `app/client/design/shared/badges.module.css`**:

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

  &.review {
    color: var(--blue);
    background: var(--blue-bg);
  }

  &.complete {
    color: var(--green);
    background: var(--green-bg);
  }

  &.uploading {
    color: var(--blue);
    background: var(--blue-bg);
  }

  &.success {
    color: var(--green);
    background: var(--green-bg);
  }

  &.error {
    color: var(--red);
    background: var(--red-bg);
  }
}
```

### Step 3: Update Existing Components for Shared Styles

**`app/client/elements/documents/document-card.ts`** — Update imports and static styles:

```typescript
import buttonStyles from '@app/design/shared/buttons.module.css';
import badgeStyles from '@app/design/shared/badges.module.css';
import styles from './document-card.module.css';

// In the class:
static styles = [buttonStyles, badgeStyles, styles];
```

**`app/client/elements/documents/document-card.module.css`** — Replace entire file (removes duplicated `.btn` and `.badge` blocks, keeps component-specific styles, uses CSS nesting):

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
}

.header {
  display: flex;
  align-items: center;
  gap: var(--space-3);
}

.filename {
  flex: 1;
  font-weight: 600;
  font-size: var(--text-sm);
  color: var(--color);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  min-width: 0;
}

.meta {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2);
  font-size: var(--text-xs);
  color: var(--color-2);

  & span:not(:last-child)::after {
    content: '\00b7';
    margin-left: var(--space-2);
    color: var(--divider);
  }
}

.classification {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  font-size: var(--text-sm);
}

.classification-label {
  font-weight: 600;
  color: var(--color);
}

.confidence {
  font-size: var(--text-xs);
  color: var(--color-1);
}

.actions {
  display: flex;
  gap: var(--space-2);
  margin-top: var(--space-1);
}

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
```

**`app/client/elements/documents/classify-progress.module.css`** — Add `prefers-reduced-motion` guard. Replace the `.stage.active .indicator` block and `@keyframes`:

```css
/* Replace the existing animation-bearing rule and @keyframes with: */

@media (prefers-reduced-motion: no-preference) {
  .stage.active .indicator {
    animation: pulse 1.5s ease-in-out infinite;
  }
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
}
```

**`app/client/components/documents/document-upload.ts`** — Update imports and static styles:

```typescript
import buttonStyles from '@app/design/shared/buttons.module.css';
import badgeStyles from '@app/design/shared/badges.module.css';
import styles from './document-upload.module.css';

// In the class:
static styles = [buttonStyles, badgeStyles, styles];
```

Also add `min="1"` to the external ID input (around line 189):

```typescript
<input
  type="number"
  class="field-input id-input"
  min="1"
  .value=${String(entry.externalId)}
  ?disabled=${!editable}
  @change=${(e: Event) => this.handleIdChange(index, e)}
/>
```

**`app/client/components/documents/document-upload.module.css`** — Replace entire file (removes duplicated `.btn` and `.badge` blocks, fixes focus styles, uses CSS nesting):

```css
:host {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

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
  transition: border-color 0.15s, background 0.15s;

  &:hover {
    border-color: var(--color-2);
    background: var(--bg-1);
  }
}

:host([dragover]) .drop-zone {
  border-color: var(--blue);
  background: var(--blue-bg);
}

.drop-icon {
  font-size: var(--text-2xl);
}

.drop-text {
  font-size: var(--text-sm);
  color: var(--color-2);
}

.queue {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  border: 1px solid var(--divider);
  border-radius: var(--radius-md);
  padding: var(--space-3);
}

.queue-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-bottom: var(--space-2);
  border-bottom: 1px solid var(--divider);
}

.queue-title {
  font-size: var(--text-sm);
  font-weight: 600;
  color: var(--color);
}

.queue-count {
  font-size: var(--text-xs);
  color: var(--color-2);
}

.queue-entry {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  padding: var(--space-2) 0;

  &:not(:last-child) {
    border-bottom: 1px solid var(--divider);
  }
}

.entry-info {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.entry-filename {
  font-size: var(--text-sm);
  font-weight: 600;
  color: var(--color);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  min-width: 0;
  flex: 1;
}

.entry-size {
  flex-shrink: 0;
  font-size: var(--text-xs);
  color: var(--color-2);
}

.entry-fields {
  display: flex;
  align-items: flex-end;
  gap: var(--space-2);
}

.field {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}

.field-label {
  font-size: var(--text-xs);
  color: var(--color-2);
}

.field-input {
  padding: var(--space-1) var(--space-2);
  border: 1px solid var(--divider);
  border-radius: var(--radius-sm);
  background: var(--bg);
  color: var(--color);
  font-size: var(--text-xs);
  font-family: var(--font-sans);

  &:focus-visible {
    outline: 2px solid var(--blue);
    outline-offset: -1px;
  }

  &:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
}

.id-input {
  width: 4rem;
}

.entry-error {
  font-size: var(--text-xs);
  color: var(--red);
}

.actions {
  display: flex;
  justify-content: space-between;
}

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

### Step 4: Pagination Pure Element

**New file: `app/client/elements/pagination/pagination-controls.ts`**:

```typescript
import { LitElement, html, nothing } from 'lit';
import { customElement, property } from 'lit/decorators.js';
import buttonStyles from '@app/design/shared/buttons.module.css';
import styles from './pagination-controls.module.css';

@customElement('hd-pagination')
export class PaginationControls extends LitElement {
  static styles = [buttonStyles, styles];

  @property({ type: Number }) page = 1;
  @property({ type: Number, attribute: 'total-pages' }) totalPages = 1;

  private handlePrev() {
    if (this.page > 1) {
      this.dispatchEvent(new CustomEvent('page-change', {
        detail: { page: this.page - 1 },
        bubbles: true,
        composed: true,
      }));
    }
  }

  private handleNext() {
    if (this.page < this.totalPages) {
      this.dispatchEvent(new CustomEvent('page-change', {
        detail: { page: this.page + 1 },
        bubbles: true,
        composed: true,
      }));
    }
  }

  render() {
    if (this.totalPages <= 1) return nothing;

    return html`
      <div class="pagination">
        <button
          class="btn"
          ?disabled=${this.page <= 1}
          @click=${this.handlePrev}
        >Prev</button>
        <span class="page-indicator">
          Page ${this.page} of ${this.totalPages}
        </span>
        <button
          class="btn"
          ?disabled=${this.page >= this.totalPages}
          @click=${this.handleNext}
        >Next</button>
      </div>
    `;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    'hd-pagination': PaginationControls;
  }
}
```

**New file: `app/client/elements/pagination/pagination-controls.module.css`**:

```css
:host {
  display: block;
}

.pagination {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: var(--space-3);
  padding-top: var(--space-3);
  border-top: 1px solid var(--divider);
}

.page-indicator {
  font-size: var(--text-sm);
  color: var(--color-1);
}
```

**New file: `app/client/elements/pagination/index.ts`**:

```typescript
export { PaginationControls } from './pagination-controls';
```

**`app/client/elements/index.ts`** — Add pagination import:

```typescript
export * from './documents';
export * from './pagination';
```

### Step 5: Document Card Selection

**`app/client/elements/documents/document-card.ts`** — Add `selected` property and `select` event handler. Update the `render()` method header to include a checkbox:

Add property:

```typescript
@property({ type: Boolean }) selected = false;
```

Add handler:

```typescript
private handleSelect() {
  this.dispatchEvent(new CustomEvent('select', {
    detail: { id: this.document.id },
    bubbles: true,
    composed: true,
  }));
}
```

Update the header in `render()`:

```typescript
<div class="header">
  <label class="select-control">
    <input
      type="checkbox"
      .checked=${this.selected}
      @change=${this.handleSelect}
    />
  </label>
  <span class="filename">${doc.filename}</span>
  <span class="badge ${doc.status}">${doc.status}</span>
</div>
```

**`app/client/elements/documents/document-card.module.css`** — Add to the file after `.header`:

```css
.select-control {
  flex-shrink: 0;
  display: flex;
  align-items: center;

  & input[type="checkbox"] {
    width: 1rem;
    height: 1rem;
    accent-color: var(--blue);
    cursor: pointer;

    &:focus-visible {
      outline: 2px solid var(--blue);
      outline-offset: 2px;
    }
  }
}
```

### Step 6: View Transitions

**New file: `app/client/view-transitions.d.ts`** — Ambient type declaration for the View Transitions API:

```typescript
interface ViewTransition {
  finished: Promise<void>;
  ready: Promise<void>;
  updateCallbackDone: Promise<void>;
}

interface Document {
  startViewTransition(callback: () => void): ViewTransition;
}
```

**`app/client/router/router.ts`** — Update `mount()` to use View Transitions API. Progressive enhancement — falls back to direct swap when unavailable:

```typescript
private mount(match: RouteMatch): void {
  const update = () => {
    this.container.innerHTML = '';
    const el = document.createElement(match.config.component);

    for (const [key, value] of Object.entries(match.params)) {
      el.setAttribute(key, value);
    }

    for (const [key, value] of Object.entries(match.query)) {
      el.setAttribute(key, value);
    }

    this.container.appendChild(el);
  };

  if (document.startViewTransition) {
    document.startViewTransition(update);
  } else {
    update();
  }
}
```

### Step 7: Document Grid Component

**New file: `app/client/components/documents/document-grid.ts`**:

```typescript
import { LitElement, html, nothing } from 'lit';
import { customElement, state } from 'lit/decorators.js';
import { DocumentService } from '@app/documents';
import type { Document, SearchRequest } from '@app/documents';
import type { PageResult } from '@app/core';
import { ClassificationService, type WorkflowStage } from '@app/classifications';
import { navigate } from '@app/router';
import buttonStyles from '@app/design/shared/buttons.module.css';
import styles from './document-grid.module.css';

interface ClassifyProgress {
  currentNode: WorkflowStage | null;
  completedNodes: WorkflowStage[];
}

@customElement('hd-document-grid')
export class DocumentGrid extends LitElement {
  static styles = [buttonStyles, styles];

  @state() private documents: PageResult<Document> | null = null;
  @state() private page = 1;
  @state() private search = '';
  @state() private status = '';
  @state() private sort = '-UploadedAt';
  @state() private classifying = new Map<string, ClassifyProgress>();
  @state() private selectedIds = new Set<string>();

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
    if (result.ok) {
      this.documents = result.data;
    }
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

  private handleStatusFilter(e: Event) {
    const select = e.target as HTMLSelectElement;
    this.status = select.value;
    this.page = 1;
    this.fetchDocuments();
  }

  private handleSort(e: Event) {
    const select = e.target as HTMLSelectElement;
    this.sort = select.value;
    this.page = 1;
    this.fetchDocuments();
  }

  private handlePageChange(e: CustomEvent<{ page: number }>) {
    this.page = e.detail.page;
    this.fetchDocuments();
  }

  private handleSelect(e: CustomEvent<{ id: string }>) {
    const id = e.detail.id;
    const next = new Set(this.selectedIds);

    if (next.has(id)) {
      next.delete(id);
    } else {
      next.add(id);
    }

    this.selectedIds = next;
  }

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
        try {
          const event = JSON.parse(data);
          const updated = new Map(this.classifying);
          const current = updated.get(docId);
          if (!current) return;

          if (type === 'node.start') {
            updated.set(docId, {
              ...current,
              currentNode: event.data?.node ?? null,
            });
          } else if (type === 'node.complete') {
            const node = event.data?.node as WorkflowStage;
            if (node) {
              updated.set(docId, {
                ...current,
                currentNode: null,
                completedNodes: [...current.completedNodes, node],
              });
            }
          }

          this.classifying = updated;
        } catch (err) {
          console.warn('Failed to parse SSE event:', data, err);
        }
      },
      onComplete: () => {
        this.abortControllers.delete(docId);
        const updated = new Map(this.classifying);
        updated.delete(docId);
        this.classifying = updated;
        this.fetchDocuments();
      },
      onError: () => {
        this.abortControllers.delete(docId);
        const updated = new Map(this.classifying);
        updated.delete(docId);
        this.classifying = updated;
        this.fetchDocuments();
      },
    });

    this.abortControllers.set(docId, controller);
  }

  private handleReview(e: CustomEvent<{ id: string }>) {
    navigate(`review/${e.detail.id}`);
  }

  private handleBulkClassify() {
    const ids = [...this.selectedIds];
    this.selectedIds = new Set();

    for (const id of ids) {
      this.handleClassify(
        new CustomEvent('classify', { detail: { id } })
      );
    }
  }

  private renderToolbar() {
    return html`
      <div class="toolbar">
        <input
          type="search"
          class="search-input"
          placeholder="Search documents..."
          .value=${this.search}
          @input=${this.handleSearchInput}
        />
        <select class="filter-select" @change=${this.handleStatusFilter}>
          <option value="">All statuses</option>
          <option value="pending" ?selected=${this.status === 'pending'}>Pending</option>
          <option value="review" ?selected=${this.status === 'review'}>Review</option>
          <option value="complete" ?selected=${this.status === 'complete'}>Complete</option>
        </select>
        <select class="sort-select" @change=${this.handleSort}>
          <option value="-UploadedAt" ?selected=${this.sort === '-UploadedAt'}>Newest</option>
          <option value="UploadedAt" ?selected=${this.sort === 'UploadedAt'}>Oldest</option>
          <option value="Filename" ?selected=${this.sort === 'Filename'}>Name A-Z</option>
          <option value="-Filename" ?selected=${this.sort === '-Filename'}>Name Z-A</option>
        </select>
        ${this.selectedIds.size > 0 ? html`
          <button
            class="btn bulk-btn"
            @click=${this.handleBulkClassify}
          >Classify ${this.selectedIds.size} selected</button>
        ` : nothing}
      </div>
    `;
  }

  private renderGrid() {
    if (!this.documents) {
      return html`<div class="empty-state">Loading...</div>`;
    }

    if (this.documents.data.length === 0) {
      return html`<div class="empty-state">No documents found.</div>`;
    }

    return html`
      <div class="grid">
        ${this.documents.data.map(doc => {
          const progress = this.classifying.get(doc.id);
          return html`
            <hd-document-card
              .document=${doc}
              ?selected=${this.selectedIds.has(doc.id)}
              ?classifying=${this.classifying.has(doc.id)}
              .currentNode=${progress?.currentNode ?? null}
              .completedNodes=${progress?.completedNodes ?? []}
              @select=${this.handleSelect}
              @classify=${this.handleClassify}
              @review=${this.handleReview}
            ></hd-document-card>
          `;
        })}
      </div>
    `;
  }

  render() {
    return html`
      ${this.renderToolbar()}
      ${this.renderGrid()}
      <hd-pagination
        .page=${this.documents?.page ?? 1}
        .totalPages=${this.documents?.total_pages ?? 1}
        @page-change=${this.handlePageChange}
      ></hd-pagination>
    `;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    'hd-document-grid': DocumentGrid;
  }
}
```

**New file: `app/client/components/documents/document-grid.module.css`**:

```css
:host {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
  flex: 1;
  min-height: 0;
}

.toolbar {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  flex-shrink: 0;
  flex-wrap: wrap;
}

.search-input,
.filter-select,
.sort-select {
  padding: var(--space-2) var(--space-3);
  border: 1px solid var(--divider);
  border-radius: var(--radius-sm);
  background: var(--bg-1);
  color: var(--color);
  font-size: var(--text-sm);
  font-family: var(--font-sans);

  &:focus-visible {
    outline: 2px solid var(--blue);
    outline-offset: 2px;
  }
}

.search-input {
  flex: 1;
  min-width: 12rem;
}

.grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: var(--space-4);
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding-bottom: var(--space-2);
}

.empty-state {
  display: flex;
  align-items: center;
  justify-content: center;
  flex: 1;
  font-size: var(--text-sm);
  color: var(--color-2);
}

.bulk-btn:not(:disabled) {
  border-color: var(--blue);
  color: var(--blue);

  &:hover {
    background: var(--blue-bg);
  }
}
```

### Step 8: Documents View Integration

**`app/client/views/documents/documents-view.ts`** — Replace entire file:

```typescript
import { LitElement, html, nothing } from 'lit';
import { customElement, state } from 'lit/decorators.js';
import buttonStyles from '@app/design/shared/buttons.module.css';
import styles from './documents-view.module.css';

@customElement('hd-documents-view')
export class DocumentsView extends LitElement {
  static styles = [buttonStyles, styles];

  @state() private showUpload = false;

  private handleUploadComplete() {
    this.showUpload = false;
    this.renderRoot.querySelector<any>('hd-document-grid')?.refresh();
  }

  render() {
    return html`
      <div class="view">
        <div class="view-header">
          <h1>Documents</h1>
          <button
            class="btn upload-toggle"
            @click=${() => this.showUpload = !this.showUpload}
          >${this.showUpload ? 'Close' : 'Upload'}</button>
        </div>
        ${this.showUpload ? html`
          <hd-document-upload
            @upload-complete=${this.handleUploadComplete}
          ></hd-document-upload>
        ` : nothing}
        <hd-document-grid></hd-document-grid>
      </div>
    `;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    'hd-documents-view': DocumentsView;
  }
}
```

**`app/client/views/documents/documents-view.module.css`** — Replace entire file:

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
  justify-content: space-between;
  flex-shrink: 0;
}

h1 {
  font-size: var(--text-xl);
  font-weight: 600;
}

.upload-toggle:not(:disabled) {
  border-color: var(--blue);
  color: var(--blue);

  &:hover {
    background: var(--blue-bg);
  }
}
```

### Step 9: Update Barrels

**`app/client/components/documents/index.ts`** — Add grid export:

```typescript
export { DocumentUpload } from './document-upload';
export { DocumentGrid } from './document-grid';
```

## Remediation

### R1: Document Create scan mismatch

Pre-existing bug: `scanDocument` scans 14 fields (11 document + 3 classification from LEFT JOIN) but `Create`'s `RETURNING` clause only returned 11 document columns. Fixed by appending `NULL, NULL, NULL` to the `RETURNING` clause in `internal/documents/repository.go` to match the scanner's expected 14 destinations.

## Validation Criteria

- [ ] `bun run build` passes
- [ ] `go vet ./...` passes
- [ ] Documents view loads at `/app/`
- [ ] Document list renders in responsive grid
- [ ] Search input filters documents with debounce
- [ ] Status filter dropdown works
- [ ] Sort control changes ordering
- [ ] Pagination controls navigate pages
- [ ] Single classify triggers SSE progress on card
- [ ] Bulk select + classify works with parallel SSE
- [ ] Upload completes and grid refreshes
- [ ] Focus-visible outlines appear on keyboard navigation
- [ ] light-dark() theme switching works
- [ ] View transitions animate route changes (Chrome/Safari)
- [ ] Pulse animation respects prefers-reduced-motion
