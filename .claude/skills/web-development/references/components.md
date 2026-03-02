# Component Patterns

Herald uses a three-tier component hierarchy. Each tier has a distinct responsibility, and crossing boundaries (e.g., a pure element calling an API) creates hidden coupling that makes components harder to test and reuse.

## View Component (provides services)

Views are route-level components. They create services and make them available to their subtree via `@provide`. `SignalWatcher` ensures the view re-renders when signal state changes.

```typescript
import { LitElement, html } from 'lit';
import { customElement } from 'lit/decorators.js';
import { provide } from '@lit/context';
import { SignalWatcher } from '@lit-labs/signals';
import { documentServiceContext, createDocumentService, DocumentService } from './service';
import styles from './documents-view.module.css';

@customElement('hd-documents-view')
export class DocumentsView extends SignalWatcher(LitElement) {
  static styles = styles;

  @provide({ context: documentServiceContext })
  private documentService: DocumentService = createDocumentService();

  connectedCallback() {
    super.connectedCallback();
    this.documentService.list();
  }

  render() {
    return html`<hd-document-list></hd-document-list>`;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    'hd-documents-view': DocumentsView;
  }
}
```

## Stateful Component (consumes services)

Stateful components receive services via `@consume` and coordinate UI interactions. They bridge the service layer with pure elements.

```typescript
import { LitElement, html } from 'lit';
import { customElement } from 'lit/decorators.js';
import { consume } from '@lit/context';
import { SignalWatcher } from '@lit-labs/signals';
import { documentServiceContext, DocumentService } from './service';
import styles from './document-list.module.css';

@customElement('hd-document-list')
export class DocumentList extends SignalWatcher(LitElement) {
  static styles = styles;

  @consume({ context: documentServiceContext })
  private documentService!: DocumentService;

  private handleDelete(e: CustomEvent<{ id: string }>) {
    this.documentService.delete(e.detail.id);
  }

  private renderDocuments() {
    return this.documentService.documents.get().map(
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
import type { Document } from '../types';
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
  const error = this.service.error.get();
  if (!error) return nothing;
  return html`<div class="error">${error}</div>`;
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
private handleSubmit(e: Event) {
  e.preventDefault();
  const form = e.target as HTMLFormElement;
  const data = new FormData(form);
  this.service.save({
    name: data.get('name') as string,
  });
}

render() {
  return html`
    <form @submit=${this.handleSubmit}>
      <input name="name" .value=${this.item.name} required />
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
