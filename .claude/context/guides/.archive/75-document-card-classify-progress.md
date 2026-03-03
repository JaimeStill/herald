# 75 - Document Card and Classify Progress Elements

## Problem Context

Issue #75 creates two pure elements for the documents view: a document card displaying document info with action buttons, and a classify progress pipeline showing SSE workflow stages. Before building them, the web-development skill's `state.md` "Encapsulated Streaming" section contradicts the three-tier hierarchy by showing a pure element calling `ClassificationService.classify()` directly. This session fixes that architectural contradiction and implements the elements with the corrected conventions.

## Architecture Approach

**Streaming orchestration belongs in the stateful component tier.** The card stays pure — props in, events out — and the list/grid component (#77) will own SSE lifecycle. The card emits `classify` and `review` events; the parent handles service calls and navigation.

**Pure element import boundary**: pure elements can import `lit`, their own CSS module, and **immutable domain infrastructure** (types, constants, formatters) from domain modules. What's prohibited is anything that holds or mutates state — services, signals, context (`@provide`/`@consume`), `SignalWatcher`. The principle: pure elements can depend on domain knowledge (what the data looks like) but not domain behavior (what the application does with it).

**Directory convention**: `elements/` directory with domain subdirectories, parallel to `views/` and `components/`.

## Implementation

### Step 1: Add `WorkflowStage` type and constant to classifications domain

**`app/client/classifications/classification.ts`** — add after the `Classification` interface:

```typescript
export type WorkflowStage = 'init' | 'classify' | 'enhance' | 'finalize';

export const WORKFLOW_STAGES: readonly WorkflowStage[] = [
  'init', 'classify', 'enhance', 'finalize',
];
```

**`app/client/classifications/index.ts`** — add the new exports:

```typescript
export type { WorkflowStage } from './classification';
export { WORKFLOW_STAGES } from './classification';
```

### Step 2: Create `hd-classify-progress` element

**`app/client/elements/documents/classify-progress.ts`** (new file)

```typescript
import { LitElement, html } from 'lit';
import { customElement, property } from 'lit/decorators.js';
import { WORKFLOW_STAGES } from '@app/classifications';
import type { WorkflowStage } from '@app/classifications';
import styles from './classify-progress.module.css';

@customElement('hd-classify-progress')
export class ClassifyProgress extends LitElement {
  static styles = styles;

  @property() currentNode: WorkflowStage | null = null;
  @property({ type: Array }) completedNodes: WorkflowStage[] = [];

  private stageState(stage: WorkflowStage): string {
    if (this.completedNodes.includes(stage)) return 'completed';
    if (stage === this.currentNode) return 'active';
    return 'pending';
  }

  render() {
    return html`
      <div class="pipeline">
        ${WORKFLOW_STAGES.map(
          (stage) => html`
            <div class="stage ${this.stageState(stage)}">
              <div class="indicator"></div>
              <span class="label">${stage}</span>
            </div>
          `
        )}
      </div>
    `;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    'hd-classify-progress': ClassifyProgress;
  }
}
```

**`app/client/elements/documents/classify-progress.module.css`** (new file)

```css
:host {
  display: block;
}

.pipeline {
  display: flex;
  position: relative;
  padding: 0 var(--space-2);
}

.pipeline::before {
  content: '';
  position: absolute;
  top: 5px;
  left: calc(var(--space-2) + 5px);
  right: calc(var(--space-2) + 5px);
  height: 2px;
  background: var(--divider);
}

.stage {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  position: relative;
  z-index: 1;
}

.indicator {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background: var(--bg-2);
  border: 2px solid var(--divider);
  transition: background 0.2s, border-color 0.2s;
}

.stage.completed .indicator {
  background: var(--green);
  border-color: var(--green);
}

.stage.active .indicator {
  background: var(--blue);
  border-color: var(--blue);
  animation: pulse 1.5s ease-in-out infinite;
}

.label {
  margin-top: var(--space-1);
  font-size: var(--text-xs);
  color: var(--color-2);
  text-transform: capitalize;
}

.stage.completed .label {
  color: var(--green);
}

.stage.active .label {
  color: var(--blue);
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
}
```

### Step 3: Create formatting utilities

**`app/client/formatting/bytes.ts`** (new file)

Mirrors Go `pkg/formatting/bytes.go` `FormatBytes` — base-1024 units.

```typescript
const units = ['B', 'KB', 'MB', 'GB', 'TB', 'PB', 'EB', 'ZB', 'YB'];

export function formatBytes(n: number, precision = 1): string {
  if (n === 0) return '0 B';

  const k = 1024;
  const i = Math.min(
    Math.floor(Math.log(n) / Math.log(k)),
    units.length - 1,
  );

  const size = n / Math.pow(k, i);
  return `${size.toFixed(Math.max(precision, 0))} ${units[i]}`;
}
```

**`app/client/formatting/date.ts`** (new file)

Locale-aware date formatting for ISO strings.

```typescript
export function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString(undefined, {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  });
}
```

**`app/client/formatting/index.ts`** (new file)

```typescript
export { formatBytes } from './bytes';
export { formatDate } from './date';
```

### Step 4: Create `hd-document-card` element

**`app/client/elements/documents/document-card.ts`** (new file)

```typescript
import { LitElement, html, nothing } from 'lit';
import { customElement, property } from 'lit/decorators.js';
import { formatBytes, formatDate } from '@app/formatting';
import type { Document } from '@app/documents';
import type { WorkflowStage } from '@app/classifications';
import styles from './document-card.module.css';

@customElement('hd-document-card')
export class DocumentCard extends LitElement {
  static styles = styles;

  @property({ type: Object }) document!: Document;
  @property({ type: Boolean }) classifying = false;
  @property() currentNode: WorkflowStage | null = null;
  @property({ type: Array }) completedNodes: WorkflowStage[] = [];

  private get classifyDisabled(): boolean {
    return this.document.status === 'complete' || this.classifying;
  }

  private handleClassify() {
    this.dispatchEvent(new CustomEvent('classify', {
      detail: { id: this.document.id },
      bubbles: true,
      composed: true,
    }));
  }

  private handleReview() {
    this.dispatchEvent(new CustomEvent('review', {
      detail: { id: this.document.id },
      bubbles: true,
      composed: true,
    }));
  }

  private renderClassification() {
    if (!this.document.classification) return nothing;

    return html`
      <div class="classification">
        <span class="classification-label">${this.document.classification}</span>
        ${this.document.confidence
          ? html`<span class="confidence">${this.document.confidence}</span>`
          : nothing}
      </div>
    `;
  }

  private renderProgress() {
    if (!this.classifying) return nothing;

    return html`
      <hd-classify-progress
        .currentNode=${this.currentNode}
        .completedNodes=${this.completedNodes}
      ></hd-classify-progress>
    `;
  }

  render() {
    const doc = this.document;

    return html`
      <div class="card">
        <div class="header">
          <span class="filename">${doc.filename}</span>
          <span class="badge ${doc.status}">${doc.status}</span>
        </div>

        <div class="meta">
          ${doc.page_count !== null
            ? html`<span>${doc.page_count} pages</span>`
            : nothing}
          <span>${formatBytes(doc.size_bytes)}</span>
          <span>${formatDate(doc.uploaded_at)}</span>
        </div>

        ${this.renderClassification()}
        ${this.renderProgress()}

        <div class="actions">
          <button
            class="btn classify-btn"
            ?disabled=${this.classifyDisabled}
            @click=${this.handleClassify}
          >Classify</button>
          <button
            class="btn review-btn"
            @click=${this.handleReview}
          >Review</button>
        </div>
      </div>
    `;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    'hd-document-card': DocumentCard;
  }
}
```

**`app/client/elements/documents/document-card.module.css`** (new file)

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
  justify-content: space-between;
  gap: var(--space-3);
}

.filename {
  font-weight: 600;
  font-size: var(--text-sm);
  color: var(--color);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  min-width: 0;
}

.badge {
  flex-shrink: 0;
  padding: var(--space-1) var(--space-2);
  border-radius: var(--radius-sm);
  font-size: var(--text-xs);
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.badge.pending {
  color: var(--yellow);
  background: var(--yellow-bg);
}

.badge.review {
  color: var(--blue);
  background: var(--blue-bg);
}

.badge.complete {
  color: var(--green);
  background: var(--green-bg);
}

.meta {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2);
  font-size: var(--text-xs);
  color: var(--color-2);
}

.meta span:not(:last-child)::after {
  content: '·';
  margin-left: var(--space-2);
  color: var(--divider);
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
}

.btn:hover:not(:disabled) {
  border-color: var(--color-2);
}

.btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.classify-btn:not(:disabled) {
  border-color: var(--blue);
  color: var(--blue);
}

.classify-btn:hover:not(:disabled) {
  background: var(--blue-bg);
}

.review-btn:not(:disabled) {
  border-color: var(--green);
  color: var(--green);
}

.review-btn:hover:not(:disabled) {
  background: var(--green-bg);
}
```

### Step 5: Barrel exports and registration

**`app/client/elements/documents/index.ts`** (new file)

```typescript
export { ClassifyProgress } from './classify-progress';
export { DocumentCard } from './document-card';
```

**`app/client/elements/index.ts`** (new file)

```typescript
export * from './documents';
```

**`app/client/app.ts`** — add elements import after the views import:

```typescript
import './elements';
```

### Step 6: Update web-development skill — `SKILL.md`

In the **Architecture Overview > Services and State** table, change the Services row description from:

> Stateless API wrappers mirroring Go handlers

to:

> Stateless API wrappers mirroring Go handlers. Called by views and stateful components only — never by pure elements.

In the **Three-Tier Component Hierarchy** table, update the Pure Element tools from:

> `@property`, `CustomEvent`

to:

> `@property`, `CustomEvent`. Imports `lit`, own CSS module, and immutable domain infrastructure (types, constants).

In the **File Structure** section, update the directory tree to include `components/` and `elements/`:

```
app/client/
├── core/                            # API layer (request, stream, types)
├── documents/                       # domain: types + service
├── classifications/                 # domain: types + service
├── prompts/                         # domain: types + service
├── storage/                         # domain: types + service
├── views/
│   └── documents/                   # view: route-level component
├── components/                      # stateful components (@consume, service calls)
│   └── documents/                   # domain-scoped components
├── elements/                        # pure elements (@property, CustomEvent)
│   └── documents/                   # domain-scoped elements
│       ├── document-card.ts
│       ├── document-card.module.css
│       ├── classify-progress.ts
│       ├── classify-progress.module.css
│       └── index.ts
├── router/
└── design/
```

After the directory tree explanation, add:

> **Component type directories** (`views/`, `components/`, `elements/`) each use domain subdirectories mirroring the domain infrastructure layout. Elements are registered via `import './elements'` in `app.ts`.

In the **Anti-Patterns > Avoid** list, update:

> - Pure elements calling services — only views and stateful components should import services

to:

> - Pure elements importing stateful infrastructure — services, signals, context (`@provide`/`@consume`), `SignalWatcher`, or router utilities. Elements can import immutable domain infrastructure (types, constants, formatters) but never anything that holds or mutates state.

In the **Anti-Patterns > Prefer** list, add:

> - Domain types and constants in pure elements — immutable domain knowledge is fine, stateful behavior is not

### Step 7: Update `references/components.md`

Replace the **Stateful Component** example with one that shows streaming orchestration:

```typescript
import { LitElement, html } from 'lit';
import { customElement, state } from 'lit/decorators.js';
import { consume } from '@lit/context';
import { SignalWatcher, Signal } from '@lit-labs/signals';
import { ClassificationService } from '@app/classifications';
import { documentsContext } from '../views/documents/documents-view';
import type { Document } from '@app/documents';
import type { PageResult } from '@app/core';
import styles from './document-list.module.css';

interface ClassifyProgress {
  currentNode: string;
  completedNodes: string[];
}

@customElement('hd-document-list')
export class DocumentList extends SignalWatcher(LitElement) {
  static styles = styles;

  @consume({ context: documentsContext })
  private documents!: Signal.State<PageResult<Document> | null>;

  @state() private classifying = new Map<string, ClassifyProgress>();

  private handleClassify(e: CustomEvent<{ id: string }>) {
    const docId = e.detail.id;
    this.classifying.set(docId, { currentNode: 'init', completedNodes: [] });
    this.requestUpdate();

    ClassificationService.classify(docId, {
      onEvent: (type, data) => {
        const event = JSON.parse(data);
        const progress = this.classifying.get(docId);
        if (!progress) return;

        switch (type) {
          case 'node.start':
            progress.currentNode = event.data.node;
            break;
          case 'node.complete':
            progress.completedNodes = [...progress.completedNodes, event.data.node];
            break;
          case 'complete':
            this.classifying.delete(docId);
            this.dispatchEvent(new CustomEvent('classify-complete', {
              bubbles: true, composed: true,
            }));
            break;
          case 'error':
            this.classifying.delete(docId);
            break;
        }
        this.requestUpdate();
      },
      onError: () => {
        this.classifying.delete(docId);
        this.requestUpdate();
      },
    });
  }

  private renderDocuments() {
    const page = this.documents.get();
    if (!page) return html`<p>Loading...</p>`;
    if (page.data.length === 0) return html`<p>No documents</p>`;

    return page.data.map((doc) => {
      const progress = this.classifying.get(doc.id);
      return html`
        <hd-document-card
          .document=${doc}
          ?classifying=${progress !== undefined}
          .currentNode=${progress?.currentNode ?? ''}
          .completedNodes=${progress?.completedNodes ?? []}
          @classify=${this.handleClassify}
        ></hd-document-card>
      `;
    });
  }

  render() {
    return html`<div class="grid">${this.renderDocuments()}</div>`;
  }
}
```

Replace the **Pure Element** example with the document card pattern:

```typescript
import { LitElement, html, nothing } from 'lit';
import { customElement, property } from 'lit/decorators.js';
import type { Document } from '@app/documents';
import type { WorkflowStage } from '@app/classifications';
import styles from './document-card.module.css';

@customElement('hd-document-card')
export class DocumentCard extends LitElement {
  static styles = styles;

  @property({ type: Object }) document!: Document;
  @property({ type: Boolean }) classifying = false;
  @property() currentNode: WorkflowStage | null = null;
  @property({ type: Array }) completedNodes: WorkflowStage[] = [];

  private handleClassify() {
    this.dispatchEvent(new CustomEvent('classify', {
      detail: { id: this.document.id },
      bubbles: true,
      composed: true,
    }));
  }

  private handleReview() {
    this.dispatchEvent(new CustomEvent('review', {
      detail: { id: this.document.id },
      bubbles: true,
      composed: true,
    }));
  }

  render() {
    return html`
      <div class="card">
        <span class="filename">${this.document.filename}</span>
        <span class="badge ${this.document.status}">${this.document.status}</span>
        ${this.classifying
          ? html`<hd-classify-progress
              .currentNode=${this.currentNode}
              .completedNodes=${this.completedNodes}
            ></hd-classify-progress>`
          : nothing}
        <button
          ?disabled=${this.document.status === 'complete' || this.classifying}
          @click=${this.handleClassify}
        >Classify</button>
        <button @click=${this.handleReview}>Review</button>
      </div>
    `;
  }
}
```

### Step 8: Update `references/state.md`

Replace the **Encapsulated Streaming** section with:

```markdown
## Streaming Orchestration

SSE operations are owned by the **stateful component** closest to the collection concern — not the pure element that triggered the action. The stateful component calls the streaming service, tracks per-item progress via `@state()`, and passes progress data to pure elements as properties.

This keeps pure elements context-free and reusable. The parent never exposes intermediate SSE events to the view — it dispatches a single completion event when the operation finishes.

\`\`\`typescript
// Stateful component — owns SSE lifecycle for the collection
@customElement('hd-document-list')
export class DocumentList extends SignalWatcher(LitElement) {
  @consume({ context: documentsContext })
  private documents!: Signal.State<PageResult<Document> | null>;

  @state() private classifying = new Map<string, ClassifyProgress>();

  private handleClassify(e: CustomEvent<{ id: string }>) {
    const docId = e.detail.id;
    this.classifying.set(docId, { currentNode: 'init', completedNodes: [] });
    this.requestUpdate();

    ClassificationService.classify(docId, {
      onEvent: (type, data) => {
        const event = JSON.parse(data);
        const progress = this.classifying.get(docId);
        if (!progress) return;

        switch (type) {
          case 'node.start':
            progress.currentNode = event.data.node;
            break;
          case 'node.complete':
            progress.completedNodes = [...progress.completedNodes, event.data.node];
            break;
          case 'complete':
            this.classifying.delete(docId);
            this.dispatchEvent(new CustomEvent('classify-complete', {
              bubbles: true, composed: true,
            }));
            break;
          case 'error':
            this.classifying.delete(docId);
            break;
        }
        this.requestUpdate();
      },
      onError: () => {
        this.classifying.delete(docId);
        this.requestUpdate();
      },
    });
  }

  render() {
    const page = this.documents.get();
    if (!page) return html\`<p>Loading...</p>\`;

    return html\`
      \${page.data.map((doc) => {
        const progress = this.classifying.get(doc.id);
        return html\`
          <hd-document-card
            .document=\${doc}
            ?classifying=\${progress !== undefined}
            .currentNode=\${progress?.currentNode ?? ''}
            .completedNodes=\${progress?.completedNodes ?? []}
            @classify=\${this.handleClassify}
          ></hd-document-card>
        \`;
      })}
    \`;
  }
}
\`\`\`

The pure element receives all streaming state as properties and dispatches intent events upward. It has no knowledge of services, SSE, or `AbortController`.

\`\`\`typescript
// Pure element — props in, events out
@customElement('hd-document-card')
export class DocumentCard extends LitElement {
  @property({ type: Object }) document!: Document;
  @property({ type: Boolean }) classifying = false;
  @property() currentNode = '';
  @property({ type: Array }) completedNodes: string[] = [];

  private handleClassify() {
    this.dispatchEvent(new CustomEvent('classify', {
      detail: { id: this.document.id },
      bubbles: true, composed: true,
    }));
  }

  render() {
    return html\`
      <button ?disabled=\${this.classifying} @click=\${this.handleClassify}>
        Classify
      </button>
      \${this.classifying
        ? html\`<hd-classify-progress .currentNode=\${this.currentNode}>\`
        : nothing}
    \`;
  }
}
\`\`\`
```

Update the **Conventions** list. Replace:

> - **Context for shared data only**: Use `@provide`/`@consume` when multiple descendants need the same reactive data

with:

> - **Context for shared data only**: Use `@provide`/`@consume` when multiple descendants need the same reactive data. Pure elements never use context — only views and stateful components.
> - **Streaming in stateful components**: The component managing the collection (list/grid) owns SSE lifecycle. Pure elements receive progress as properties and dispatch intent events.

Remove the standalone "Encapsulated Streaming" heading and its obsolete example showing the card calling `ClassificationService.classify()`.

## Validation Criteria

- [ ] `hd-classify-progress` renders 4-stage pipeline with pending/active/completed visual states
- [ ] `hd-document-card` renders filename, page count, date, size, status badge
- [ ] Status badge visually differentiates pending/review/complete via color tokens
- [ ] Classification summary shown conditionally when document has classification data
- [ ] Classify button dispatches `classify` CustomEvent with document ID
- [ ] Classify button disabled when status === 'complete' or classifying is true
- [ ] Review button dispatches `review` CustomEvent with document ID
- [ ] Progress element shown inline when classifying is true
- [ ] Neither element imports stateful infrastructure (services, signals, context, router)
- [ ] `WorkflowStage` type and `WORKFLOW_STAGES` constant exported from classifications domain
- [ ] Both elements use `*.module.css` with design tokens
- [ ] `app.ts` imports `./elements` for registration
- [ ] Skill references updated — no more contradiction between tiers and streaming
- [ ] `bun run build` compiles without errors
