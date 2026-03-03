# Shared Reactive State

Views and stateful components use `Signal.State` signals with `@lit/context` to share reactive data across their subtree. Components call services directly and update signals themselves — there is no orchestration layer between services and components.

## When to Use Context

Use `@provide`/`@consume` with signals when a parent component has data that multiple descendants need reactively. If only one child needs the data, pass it as a `@property` instead.

```typescript
// View provides shared data to its subtree
@provide({ context: documentsContext })
private documents = new Signal.State<PageResult<Document> | null>(null);
```

## Providing Shared State

Views create signals as class fields and `@provide` them. The view calls services directly and updates its own signals.

```typescript
import { LitElement, html } from 'lit';
import { customElement } from 'lit/decorators.js';
import { provide } from '@lit/context';
import { SignalWatcher, Signal } from '@lit-labs/signals';
import { createContext } from '@lit/context';
import { DocumentService } from '@app/documents';
import type { Document, PageResult, PageRequest } from '@app/documents';

export const documentsContext =
  createContext<Signal.State<PageResult<Document> | null>>('documents');

@customElement('hd-documents-view')
export class DocumentsView extends SignalWatcher(LitElement) {
  @provide({ context: documentsContext })
  private documents = new Signal.State<PageResult<Document> | null>(null);

  async connectedCallback() {
    super.connectedCallback();
    const result = await DocumentService.list();
    if (result.ok) this.documents.set(result.data);
  }

  render() {
    return html`<hd-document-list></hd-document-list>`;
  }
}
```

## Consuming Shared State

Descendants `@consume` the signal and call `.get()` in templates. `SignalWatcher` drives re-renders when the signal value changes.

```typescript
import { LitElement, html } from 'lit';
import { customElement } from 'lit/decorators.js';
import { consume } from '@lit/context';
import { SignalWatcher, Signal } from '@lit-labs/signals';
import { documentsContext } from '../views/documents/documents-view';
import type { Document, PageResult } from '@app/documents';

@customElement('hd-document-list')
export class DocumentList extends SignalWatcher(LitElement) {
  @consume({ context: documentsContext })
  private documents!: Signal.State<PageResult<Document> | null>;

  render() {
    const page = this.documents.get();
    if (!page) return html`<p>Loading...</p>`;
    return html`
      ${page.data.map((doc) => html`
        <hd-document-card .document=${doc}></hd-document-card>
      `)}
    `;
  }
}
```

## Signal Initialization

- `null` means "not yet loaded" — components show skeleton/spinner
- Empty `PageResult` (`data: []`) means "loaded but empty" — components show empty state
- Pagination metadata comes from the server, never hardcoded on the client

```typescript
const documents = new Signal.State<PageResult<Document> | null>(null);
```

## Context Key Conventions

- kebab-case strings: `'documents'`, `'prompts'`, `'loading'`
- Defined adjacent to the providing component, not in a shared file
- Typed via the generic: `createContext<Signal.State<T>>('key')`

## Components Call Services Directly

There is no orchestration layer between services and components. Components import services, call them, and update their own state (signals or `@state()`) based on results.

```typescript
// View calls service and updates its signal
private async refresh(params?: PageRequest) {
  const result = await DocumentService.list(params);
  if (result.ok) this.documents.set(result.data);
}

// Stateful component calls service for its own concern
private async handleDelete(id: string) {
  const result = await DocumentService.delete(id);
  if (result.ok) {
    this.dispatchEvent(new CustomEvent('document-deleted', {
      detail: { id },
      bubbles: true,
      composed: true,
    }));
  }
}
```

## Streaming Orchestration

SSE operations are owned by the **stateful component** closest to the collection concern — not the pure element that triggered the action. The stateful component calls the streaming service, tracks per-item progress via `@state()`, and passes progress data to pure elements as properties.

This keeps pure elements context-free and reusable. The parent dispatches a single completion event when the operation finishes — the view never sees intermediate progress events.

```typescript
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
    if (!page) return html`<p>Loading...</p>`;

    return html`
      ${page.data.map((doc) => {
        const progress = this.classifying.get(doc.id);
        return html`
          <hd-document-card
            .document=${doc}
            ?classifying=${progress !== undefined}
            .currentNode=${progress?.currentNode ?? null}
            .completedNodes=${progress?.completedNodes ?? []}
            @classify=${this.handleClassify}
          ></hd-document-card>
        `;
      })}
    `;
  }
}
```

The pure element receives all streaming state as properties and dispatches intent events upward. It has no knowledge of services, SSE, or `AbortController`.

```typescript
// Pure element — props in, events out
@customElement('hd-document-card')
export class DocumentCard extends LitElement {
  @property({ type: Object }) document!: Document;
  @property({ type: Boolean }) classifying = false;
  @property() currentNode: WorkflowStage | null = null;
  @property({ type: Array }) completedNodes: WorkflowStage[] = [];

  private handleClassify() {
    this.dispatchEvent(new CustomEvent('classify', {
      detail: { id: this.document.id },
      bubbles: true, composed: true,
    }));
  }

  render() {
    return html`
      <button
        ?disabled=${this.document.status === 'complete' || this.classifying}
        @click=${this.handleClassify}
      >Classify</button>
      ${this.classifying
        ? html`<hd-classify-progress
            .currentNode=${this.currentNode}
            .completedNodes=${this.completedNodes}
          ></hd-classify-progress>`
        : nothing}
    `;
  }
}
```

## Conventions

- **Context for shared data only**: Use `@provide`/`@consume` when multiple descendants need the same reactive data. Pure elements never use context — only views and stateful components.
- **Streaming in stateful components**: The component managing the collection (list/grid) owns SSE lifecycle. Pure elements receive progress as properties and dispatch intent events.
- **`@state()` for local concerns**: Classification progress, form errors, UI toggles — use Lit's built-in reactive state
- **Views and stateful components call services directly**: No orchestration middleman between services and components. Pure elements dispatch events instead.
- **Events up, data down**: Children dispatch `CustomEvent` to notify parents of outcomes
- **Signal reads**: `.get()` in templates inside `SignalWatcher` components
- **Signal writes**: `.set()` in the component that owns the signal
- **No state files**: State is defined inline in the component that provides it
