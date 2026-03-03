# Plan: #74 — Document types, service, and SSE stream enhancement

## Context

First sub-issue of Objective #59 (Document Management View). Establishes the data layer that all subsequent view components depend on: enhanced SSE streaming, TypeScript domain types mirroring Go API shapes, and a reactive service with Signal.State signals. No existing callers of `stream()` beyond its export — the breaking signature change is safe.

This task also establishes the client-side domain directory convention: domain infrastructure (types, services) live in their own directories under `app/client/`, separate from view components in `views/`. The web-development skill and project context documents will be updated to capture these conventions.

## Directory Structure (New Convention)

```
app/client/
├── core/
│   ├── api.ts              # stream() enhancement + ExecutionEvent type
│   └── index.ts            # barrel (re-export ExecutionEvent)
├── documents/
│   ├── document.ts         # Document type + DocumentStatus
│   ├── service.ts          # DocumentService + context + factory
│   └── index.ts            # barrel
├── classifications/
│   ├── classification.ts   # Classification type
│   └── index.ts            # barrel
├── components/             # (future) stateful components
├── elements/               # (future) stateless/pure elements
├── views/
│   ├── documents/          # view components only (no types/services)
│   │   ├── documents-view.ts
│   │   ├── documents-view.module.css
│   │   └── index.ts
│   ...
```

**Convention**: Domain directories (`documents/`, `classifications/`) hold types and services. `views/` holds view components that import from domain directories. `components/` holds stateful components, `elements/` holds pure elements. View subdirectories may contain view-specific sub-components if tightly coupled.

## Files Modified

| File | Action |
|------|--------|
| `app/client/core/api.ts` | Modify — enhance `stream()`, add `ExecutionEvent` type |
| `app/client/core/index.ts` | Modify — re-export `ExecutionEvent` |
| `app/client/documents/document.ts` | Create — `Document`, `DocumentStatus` |
| `app/client/documents/service.ts` | Create — `DocumentService`, context, factory |
| `app/client/documents/index.ts` | Create — barrel |
| `app/client/classifications/classification.ts` | Create — `Classification` |
| `app/client/classifications/index.ts` | Create — barrel |
| `app/client/views/documents/index.ts` | Verify — should only export view component |

## Step 1: Enhance `stream()` and add `ExecutionEvent` in `core/api.ts`

### ExecutionEvent type

Add at the top of `api.ts` alongside other types:

```typescript
export interface ExecutionEvent {
  type: string;
  timestamp: string;
  data: Record<string, unknown>;
}
```

This lives in `core/` because it's SSE streaming infrastructure, not domain-specific.

### StreamOptions changes

Replace `onMessage` callback with `onEvent`:

```typescript
export interface StreamOptions {
  onEvent: (type: string, data: string) => void;
  onError?: (error: string) => void;
  onComplete?: () => void;
  signal?: AbortSignal;
}
```

### stream() signature

Add `init?: RequestInit` as third parameter:

```typescript
export function stream(
  path: string,
  options: StreamOptions,
  init?: RequestInit,
): AbortController
```

### Parsing logic

Track current event type from `event:` lines, pair with next `data:` line:

```typescript
let buffer = '';
let currentEvent = 'message'; // default SSE event type

for (const line of lines) {
  if (line.startsWith('event: ')) {
    currentEvent = line.slice(7).trim();
  } else if (line.startsWith('data: ')) {
    const data = line.slice(6).trim();
    options.onEvent(currentEvent, data);
    currentEvent = 'message'; // reset to default
  }
}
```

### Other changes

- Remove `[DONE]` sentinel check — completion signaled when reader reports `done`
- Merge init with signal: `fetch(\`${BASE}${path}\`, { ...init, signal })`

### Update `core/index.ts`

Add `ExecutionEvent` to the type re-export line.

## Step 2: Document type in `documents/document.ts`

```typescript
export type DocumentStatus = 'pending' | 'review' | 'complete';

export interface Document {
  id: string;
  external_id: number;
  external_platform: string;
  filename: string;
  content_type: string;
  size_bytes: number;
  page_count: number | null;
  storage_key: string;
  status: DocumentStatus;
  uploaded_at: string;
  updated_at: string;
  classification?: string;
  confidence?: string;
  classified_at?: string;
}
```

Mirrors Go `documents.Document` struct. `page_count` is `number | null` (Go `*int`, not omitempty). Optional fields match `omitempty` tags.

## Step 3: Classification type in `classifications/classification.ts`

```typescript
export interface Classification {
  id: string;
  document_id: string;
  classification: string;
  confidence: string;
  markings_found: string[];
  rationale: string;
  classified_at: string;
  model_name: string;
  provider_name: string;
  validated_by?: string;
  validated_at?: string;
}
```

Mirrors Go `classifications.Classification` struct.

## Step 4: DocumentService in `documents/service.ts`

Follows the established service pattern (Signal.State + @lit/context + factory).

```typescript
import { createContext } from '@lit/context';
import { Signal } from '@lit-labs/signals';
import {
  request, stream, toQueryString,
  type PageResult, type PageRequest,
} from '@app/core';
import type { Document } from './document';

export interface DocumentService {
  documents: Signal.State<PageResult<Document>>;
  loading: Signal.State<boolean>;
  error: Signal.State<string | null>;
  classifyingIds: Signal.State<Set<string>>;

  list(params?: PageRequest): void;
  upload(file: File, externalId: number, platform: string): void;
  remove(id: string): void;
  classify(documentId: string): void;
}

export const documentServiceContext =
  createContext<DocumentService>('document-service');

export function createDocumentService(): DocumentService { ... }
```

### Signal initialization

```typescript
const documents = new Signal.State<PageResult<Document>>({
  data: [], total: 0, page: 1, page_size: 20, total_pages: 0,
});
const loading = new Signal.State(false);
const error = new Signal.State<string | null>(null);
const classifyingIds = new Signal.State<Set<string>>(new Set());
```

`PageResult<Document>` carries pagination metadata — views access `.get().data` for the list.

### Method implementations

- **list(params?)**: GET `/documents` with query string. Sets `loading`, updates `documents` with full `PageResult`, clears on error.
- **upload(file, externalId, platform)**: POST `/documents` with FormData. On success, refreshes the list to pick up the new document with correct pagination.
- **remove(id)**: DELETE `/documents/${id}`. On success, refreshes the list.
- **classify(documentId)**: POST `/classifications/${documentId}` via `stream()`. Adds ID to `classifyingIds`, processes SSE events, removes from `classifyingIds` on complete/error. On `complete` event, refreshes the document list.

### classifyingIds management

Must create new Set for signal reactivity (mutation doesn't trigger):

```typescript
const ids = new Set(classifyingIds.get());
ids.add(documentId);
classifyingIds.set(ids);
```

## Step 5: Create barrel exports

### `documents/index.ts`

```typescript
export * from './document';
export * from './service';
```

### `classifications/index.ts`

```typescript
export * from './classification';
```

## Step 6: Update web-development skill and context docs

Update `.claude/skills/web-development/SKILL.md` and relevant references to capture:

1. **Domain directory convention**: Types and services in domain directories (`documents/`, `classifications/`), separate from views
2. **Component organization**: `components/` for stateful, `elements/` for pure/stateless
3. **View directories**: Only view components in `views/<domain>/`, importing from domain directories

Update `_project/objective.md` if any structural decisions need capturing.

## Validation Criteria

- [ ] `stream()` accepts `init?: RequestInit` and merges with abort signal
- [ ] `stream()` parses `event:` lines and pairs with `data:` lines via `onEvent(type, data)` callback
- [ ] `stream()` completes when reader reports done (no `[DONE]` sentinel dependency)
- [ ] TypeScript types match Go API JSON response shapes
- [ ] `ExecutionEvent` lives in `core/api.ts`, re-exported from `core/index.ts`
- [ ] `Document` and `DocumentStatus` in `documents/document.ts`
- [ ] `Classification` in `classifications/classification.ts`
- [ ] `DocumentService` in `documents/service.ts` with all four signals and four methods
- [ ] `documentServiceContext` exported for `@provide`/`@consume`
- [ ] Domain barrels export types and services
- [ ] Web-development skill updated with domain directory conventions
- [ ] `bun run build` compiles without errors
