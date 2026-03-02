# Service Infrastructure

Each domain has a single `service.ts` that exports three things: a context key, an interface, and a factory function. This co-location keeps the service contract, creation, and dependency injection in one place.

## Consolidated Service File

```typescript
// documents/service.ts
import { createContext } from '@lit/context';
import { Signal } from '@lit-labs/signals';
import { request, type PageResult, toQueryString, type PageRequest } from '@app/core';

export interface DocumentService {
  documents: Signal.State<Document[]>;
  loading: Signal.State<boolean>;
  error: Signal.State<string | null>;

  list(params?: PageRequest): void;
  find(id: string): void;
  upload(file: File): void;
  delete(id: string): void;
}

export const documentServiceContext = createContext<DocumentService>('document-service');

export function createDocumentService(): DocumentService {
  const documents = new Signal.State<Document[]>([]);
  const loading = new Signal.State<boolean>(false);
  const error = new Signal.State<string | null>(null);

  return {
    documents,
    loading,
    error,

    async list(params) {
      loading.set(true);
      const qs = params ? toQueryString(params) : '';
      const result = await request<PageResult<Document>>(`/documents${qs}`);
      if (result.ok) {
        documents.set(result.data.data);
      } else {
        error.set(result.error);
      }
      loading.set(false);
    },

    async find(id) { /* ... */ },
    async upload(file) { /* ... */ },
    async delete(id) { /* ... */ },
  };
}
```

## How Services Flow Through the Hierarchy

1. **View creates** the service via factory and `@provide`s it:

```typescript
@provide({ context: documentServiceContext })
private documentService: DocumentService = createDocumentService();
```

2. **Stateful components** `@consume` the service from context:

```typescript
@consume({ context: documentServiceContext })
private documentService!: DocumentService;
```

3. **Pure elements** never touch services — they receive data via `@property` and emit events upward.

## Signal Reactivity

`Signal.State` values are read with `.get()` and written with `.set()`. The `SignalWatcher` mixin on view and stateful components ensures Lit re-renders when any signal read during `render()` changes.

Without `SignalWatcher`, signal changes will not trigger re-renders — this is a common mistake.

```typescript
// Reading in templates (inside SignalWatcher component)
${this.service.loading.get() ? html`<p>Loading...</p>` : nothing}

// Writing in service methods
loading.set(true);
documents.set(result.data.data);
```

## Service Conventions

- One `service.ts` per domain (documents, prompts, classifications)
- Context string keys are kebab-case: `'document-service'`, `'prompt-service'`
- Factory functions are `createXxxService()` — pure functions, no side effects until called
- Signals expose `.get()` for reading and `.set()` for writing
- Error state is `Signal.State<string | null>` — `null` means no error
- Loading state gates UI to prevent stale data display
