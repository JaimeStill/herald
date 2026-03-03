# Component Patterns

Herald uses a three-tier component hierarchy. Each tier has a distinct responsibility, and crossing boundaries (e.g., a pure element calling an API) creates hidden coupling that makes components harder to test and reuse.

## View Component (provides shared state)

Views are route-level components. They call services, manage shared reactive state via `Signal.State`, and `@provide` it to their subtree. `SignalWatcher` ensures the view re-renders when signal state changes.

```typescript
import { LitElement, html } from 'lit';
import { customElement } from 'lit/decorators.js';
import { provide } from '@lit/context';
import { createContext } from '@lit/context';
import { SignalWatcher, Signal } from '@lit-labs/signals';
import { DocumentService } from '@app/documents';
import type { Document } from '@app/documents';
import type { PageResult, PageRequest } from '@app/core';
import styles from './documents-view.module.css';

export const documentsContext =
  createContext<Signal.State<PageResult<Document> | null>>('documents');

@customElement('hd-documents-view')
export class DocumentsView extends SignalWatcher(LitElement) {
  static styles = styles;

  @provide({ context: documentsContext })
  private documents = new Signal.State<PageResult<Document> | null>(null);

  async connectedCallback() {
    super.connectedCallback();
    await this.refresh();
  }

  private async refresh(params?: PageRequest) {
    const result = await DocumentService.list(params);
    if (result.ok) this.documents.set(result.data);
  }

  private handleDocumentDeleted() {
    this.refresh();
  }

  private handleClassifyComplete() {
    this.refresh();
  }

  render() {
    return html`
      <hd-document-list
        @document-deleted=${this.handleDocumentDeleted}
        @classify-complete=${this.handleClassifyComplete}
      ></hd-document-list>
    `;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    'hd-documents-view': DocumentsView;
  }
}
```

## Stateful Component (consumes state, calls services)

Stateful components receive shared state via `@consume` and call services directly for their own concerns. They bridge shared state with pure elements and manage local UI state with `@state()`.

```typescript
import { LitElement, html } from 'lit';
import { customElement, state } from 'lit/decorators.js';
import { consume } from '@lit/context';
import { SignalWatcher, Signal } from '@lit-labs/signals';
import { DocumentService } from '@app/documents';
import { ClassificationService } from '@app/classifications';
import { documentsContext } from '../views/documents/documents-view';
import type { Document } from '@app/documents';
import type { PageResult } from '@app/core';
import styles from './document-list.module.css';

@customElement('hd-document-list')
export class DocumentList extends SignalWatcher(LitElement) {
  static styles = styles;

  @consume({ context: documentsContext })
  private documents!: Signal.State<PageResult<Document> | null>;

  private async handleDelete(e: CustomEvent<{ id: string }>) {
    const result = await DocumentService.delete(e.detail.id);
    if (result.ok) {
      this.dispatchEvent(new CustomEvent('document-deleted', {
        bubbles: true,
        composed: true,
      }));
    }
  }

  private renderDocuments() {
    const page = this.documents.get();
    if (!page) return html`<p>Loading...</p>`;
    if (page.data.length === 0) return html`<p>No documents</p>`;

    return page.data.map(
      (doc) => html`
        <hd-document-card
          .document=${doc}
          @delete=${this.handleDelete}
        ></hd-document-card>
      `
    );
  }

  render() {
    return html`<div class="grid">${this.renderDocuments()}</div>`;
  }
}
```

## Pure Element (stateless)

Pure elements receive data via properties and communicate upward through events. They have no knowledge of services or application state.

```typescript
import { LitElement, html } from 'lit';
import { customElement, property } from 'lit/decorators.js';
import type { Document } from '@app/documents';
import styles from './document-card.module.css';

@customElement('hd-document-card')
export class DocumentCard extends LitElement {
  static styles = styles;

  @property({ type: Object }) document!: Document;

  private handleDelete() {
    this.dispatchEvent(new CustomEvent('delete', {
      detail: { id: this.document.id },
      bubbles: true,
      composed: true,
    }));
  }

  render() {
    return html`
      <div class="card">
        <h3>${this.document.filename}</h3>
        <p>${this.document.status}</p>
        <button @click=${this.handleDelete}>Delete</button>
      </div>
    `;
  }
}
```

## Template Patterns

### Render Methods

Extract complex template logic into private `renderXxx()` methods. Use `nothing` from Lit for conditional non-rendering — it produces no DOM output, unlike an empty string which creates a text node.

```typescript
import { nothing } from 'lit';

private renderError() {
  if (!this.error) return nothing;
  return html`<div class="error">${this.error}</div>`;
}

render() {
  return html`
    ${this.renderError()}
    ${this.renderContent()}
  `;
}
```

### Form Handling

Extract values via FormData on submit rather than tracking controlled inputs:

```typescript
private async handleSubmit(e: Event) {
  e.preventDefault();
  const form = e.target as HTMLFormElement;
  const data = new FormData(form);

  const result = await PromptService.create({
    name: data.get('name') as string,
    stage: data.get('stage') as PromptStage,
    instructions: data.get('instructions') as string,
  });

  if (result.ok) {
    this.dispatchEvent(new CustomEvent('prompt-created', {
      bubbles: true, composed: true,
    }));
  }
}

render() {
  return html`
    <form @submit=${this.handleSubmit}>
      <input name="name" required />
      <button type="submit">Save</button>
    </form>
  `;
}
```

### Host Attribute Reflection

Reflect component state to the host element so CSS can drive layout changes without JavaScript:

```typescript
@state() private expanded = false;

updated(changed: Map<string, unknown>) {
  if (changed.has('expanded')) {
    this.toggleAttribute('expanded', this.expanded);
  }
}
```

```css
:host { grid-template-rows: auto auto 1fr; }
:host([expanded]) { grid-template-rows: auto 1fr 1fr; }
```

### Object URL Lifecycle

Revoke blob URLs in `disconnectedCallback` to prevent memory leaks:

```typescript
private imageUrls = new Map<File, string>();

disconnectedCallback() {
  super.disconnectedCallback();
  this.imageUrls.forEach((url) => URL.revokeObjectURL(url));
  this.imageUrls.clear();
}

private getImageUrl(file: File): string {
  let url = this.imageUrls.get(file);
  if (!url) {
    url = URL.createObjectURL(file);
    this.imageUrls.set(file, url);
  }
  return url;
}
```
