# 74 - Document types, service, and SSE stream enhancement

## Problem Context

First sub-issue of Objective #59 (Document Management View). The web client currently has placeholder views with no data layer. This task establishes the foundational infrastructure that all subsequent view components depend on: enhanced SSE streaming to support the classification endpoint's `event:` + `data:` format with POST method, TypeScript domain types mirroring Go API response shapes, and stateless services that mirror the Go API domain architecture.

This also establishes two key client-side conventions:

1. **Domain directory convention**: Types and services live in their own domain directories (`documents/`, `classifications/`) separate from view components (`views/`).
2. **Service/state separation**: Services are stateless API wrappers that mirror Go domains. State orchestration (signals, context, reactive coordination) lives adjacent to the views/components that need it and is delivered in later sub-issues.

## Architecture Approach

**Services** — stateless objects that mirror Go API domain structure. Each service has a `base` path constant and methods that map directly to server endpoints. Methods return `Result<T>` for request-response, `AbortController` for streaming. No signals, no context, no state.

**Types** — domain interfaces in their own directories, mirroring Go struct shapes. `ExecutionEvent` lives in `core/` as SSE streaming infrastructure.

**State** — deferred to view assembly tasks (#75–#77). Components call services directly and manage their own state. Shared reactive data (e.g., document list) uses `Signal.State` via `@lit/context`. Local concerns (classify progress, errors) use `@state()`. No state orchestration layer between services and components.

## Implementation

### Step 1: Enhance `stream()` and add `ExecutionEvent` in `core/api.ts`

In `app/client/core/api.ts`, add the `ExecutionEvent` interface after the `Result` type:

```typescript
export interface ExecutionEvent {
  type: string;
  timestamp: string;
  data: Record<string, unknown>;
}
```

Replace the `StreamOptions` interface:

```typescript
export interface StreamOptions {
  onEvent: (type: string, data: string) => void;
  onError?: (error: string) => void;
  onComplete?: () => void;
  signal?: AbortSignal;
}
```

Replace the `stream()` function:

```typescript
export function stream(
  path: string,
  options: StreamOptions,
  init?: RequestInit,
): AbortController {
  const controller = new AbortController();
  const signal = options.signal ?? controller.signal;

  fetch(`${BASE}${path}`, { ...init, signal })
    .then(async (res) => {
      if (!res.ok) {
        const text = await res.text();
        options.onError?.(text || res.statusText);
        return;
      }

      const reader = res.body?.getReader();
      if (!reader) {
        options.onError?.('No response body');
        return;
      }

      const decoder = new TextDecoder();
      let buffer = '';
      let currentEvent = 'message';

      while (true) {
        const { done, value } = await reader.read();
        if (done) break;

        buffer += decoder.decode(value, { stream: true });
        const lines = buffer.split('\n');
        buffer = lines.pop() ?? '';

        for (const line of lines) {
          if (line.startsWith('event: ')) {
            currentEvent = line.slice(7).trim();
          } else if (line.startsWith('data: ')) {
            const data = line.slice(6).trim();
            options.onEvent(currentEvent, data);
            currentEvent = 'message';
          }
        }
      }

      options.onComplete?.();
    })
    .catch((err: Error) => {
      if (err.name !== 'AbortError') {
        options.onError?.(err.message);
      }
    });

  return controller;
}
```

Key changes from existing code:
- `onMessage` → `onEvent(type, data)` with event type tracking
- Added `init?: RequestInit` parameter, merged via `{ ...init, signal }`
- Removed `[DONE]` sentinel — completion on reader `done`
- Tracks `currentEvent` from `event:` lines, pairs with next `data:` line, resets to `'message'`

### Step 2: Update `core/index.ts` barrel

In `app/client/core/index.ts`, add `ExecutionEvent` to the type re-export:

```typescript
export { request, stream, toQueryString } from './api';
export type { Result, StreamOptions, ExecutionEvent, PageResult, PageRequest } from './api';
```

### Step 3: Create `documents/document.ts`

Create `app/client/documents/document.ts`:

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

### Step 4: Create `classifications/classification.ts`

Create `app/client/classifications/classification.ts`:

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

### Step 5: Create `prompts/prompt.ts`

Create `app/client/prompts/prompt.ts`:

```typescript
export type PromptStage = 'classify' | 'enhance' | 'finalize';

export interface Prompt {
  id: string;
  name: string;
  stage: PromptStage;
  instructions: string;
  description?: string;
  active: boolean;
}

export interface StageContent {
  stage: PromptStage;
  content: string;
}

export interface CreatePromptCommand {
  name: string;
  stage: PromptStage;
  instructions: string;
  description?: string;
}

export interface UpdatePromptCommand {
  name: string;
  stage: PromptStage;
  instructions: string;
  description?: string;
}
```

### Step 6: Create `storage/blob.ts`

Create `app/client/storage/blob.ts`:

```typescript
export interface BlobMeta {
  name: string;
  content_type: string;
  content_length: number;
  last_modified: string;
  etag: string;
  created_at: string;
}

export interface BlobList {
  blobs: BlobMeta[];
  next_marker?: string;
}
```

### Step 7: Create `documents/service.ts`

Create `app/client/documents/service.ts`:

Stateless service object mirroring the Go `documents` handler. Every handler method has a corresponding service method. Base path captured once. Returns `Result<T>` — no signals, no state.

```typescript
import {
  request, toQueryString,
  type Result, type PageResult, type PageRequest,
} from '@app/core';
import type { Document } from './document';

const base = '/documents';

export const DocumentService = {
  async list(params?: PageRequest): Promise<Result<PageResult<Document>>> {
    return await request<PageResult<Document>>(
      `${base}${params ? toQueryString(params) : ''}`
    );
  },

  async find(id: string): Promise<Result<Document>> {
    return await request<Document>(`${base}/${id}`);
  },

  async search(body: PageRequest): Promise<Result<PageResult<Document>>> {
    return await request<PageResult<Document>>(`${base}/search`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    });
  },

  async upload(
    file: File,
    externalId: number,
    platform: string,
  ): Promise<Result<Document>> {
    const form = new FormData();
    form.append('file', file);
    form.append('external_id', String(externalId));
    form.append('external_platform', platform);

    return await request<Document>(base, {
      method: 'POST',
      body: form,
    });
  },

  async delete(id: string): Promise<Result<void>> {
    return await request<void>(`${base}/${id}`, {
      method: 'DELETE',
    });
  },
};
```

### Step 6: Create `classifications/service.ts`

Create `app/client/classifications/service.ts`:

Stateless service object mirroring the Go `classifications` handler. The `classify` method returns the `AbortController` from `stream()` — the caller provides `StreamOptions` callbacks, giving the state layer full control over SSE event processing.

```typescript
import {
  request, stream, toQueryString,
  type Result, type PageResult, type PageRequest, type StreamOptions,
} from '@app/core';
import type { Classification } from './classification';

export interface ValidateCommand {
  validated_by: string;
}

export interface UpdateCommand {
  classification: string;
  rationale: string;
  updated_by: string;
}

const base = '/classifications';

export const ClassificationService = {
  async list(params?: PageRequest): Promise<Result<PageResult<Classification>>> {
    return await request<PageResult<Classification>>(
      `${base}${params ? toQueryString(params) : ''}`
    );
  },

  async find(id: string): Promise<Result<Classification>> {
    return await request<Classification>(`${base}/${id}`);
  },

  async findByDocument(documentId: string): Promise<Result<Classification>> {
    return await request<Classification>(`${base}/document/${documentId}`);
  },

  async search(body: PageRequest): Promise<Result<PageResult<Classification>>> {
    return await request<PageResult<Classification>>(`${base}/search`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    });
  },

  classify(documentId: string, options: StreamOptions): AbortController {
    return stream(`${base}/${documentId}`, options, { method: 'POST' });
  },

  async validate(id: string, command: ValidateCommand): Promise<Result<Classification>> {
    return await request<Classification>(`${base}/${id}/validate`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(command),
    });
  },

  async update(id: string, command: UpdateCommand): Promise<Result<Classification>> {
    return await request<Classification>(`${base}/${id}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(command),
    });
  },

  async delete(id: string): Promise<Result<void>> {
    return await request<void>(`${base}/${id}`, {
      method: 'DELETE',
    });
  },
};
```

### Step 8: Create `prompts/service.ts`

Create `app/client/prompts/service.ts`:

Stateless service object mirroring the Go `prompts` handler. All 11 handler methods mapped.

```typescript
import {
  request, toQueryString,
  type Result, type PageResult, type PageRequest,
} from '@app/core';
import type { Prompt, StageContent, PromptStage, CreatePromptCommand, UpdatePromptCommand } from './prompt';

const base = '/prompts';

export const PromptService = {
  async list(params?: PageRequest): Promise<Result<PageResult<Prompt>>> {
    return await request<PageResult<Prompt>>(
      `${base}${params ? toQueryString(params) : ''}`
    );
  },

  async stages(): Promise<Result<PromptStage[]>> {
    return await request<PromptStage[]>(`${base}/stages`);
  },

  async find(id: string): Promise<Result<Prompt>> {
    return await request<Prompt>(`${base}/${id}`);
  },

  async instructions(stage: PromptStage): Promise<Result<StageContent>> {
    return await request<StageContent>(`${base}/${stage}/instructions`);
  },

  async spec(stage: PromptStage): Promise<Result<StageContent>> {
    return await request<StageContent>(`${base}/${stage}/spec`);
  },

  async create(command: CreatePromptCommand): Promise<Result<Prompt>> {
    return await request<Prompt>(base, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(command),
    });
  },

  async update(id: string, command: UpdatePromptCommand): Promise<Result<Prompt>> {
    return await request<Prompt>(`${base}/${id}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(command),
    });
  },

  async delete(id: string): Promise<Result<void>> {
    return await request<void>(`${base}/${id}`, {
      method: 'DELETE',
    });
  },

  async search(body: PageRequest): Promise<Result<PageResult<Prompt>>> {
    return await request<PageResult<Prompt>>(`${base}/search`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    });
  },

  async activate(id: string): Promise<Result<Prompt>> {
    return await request<Prompt>(`${base}/${id}/activate`, {
      method: 'POST',
    });
  },

  async deactivate(id: string): Promise<Result<Prompt>> {
    return await request<Prompt>(`${base}/${id}/deactivate`, {
      method: 'POST',
    });
  },
};
```

### Step 9: Create `storage/service.ts`

Create `app/client/storage/service.ts`:

Stateless service object mirroring the Go `storage` handler. The `download` method uses a custom `parse` callback to return a `Blob` instead of JSON, and constructs a download URL for direct browser use.

```typescript
import {
  request,
  type Result,
} from '@app/core';
import type { BlobMeta, BlobList } from './blob';

export interface StorageListParams {
  prefix?: string;
  marker?: string;
  max_results?: number;
}

const base = '/storage';

export const StorageService = {
  async list(params?: StorageListParams): Promise<Result<BlobList>> {
    const entries = Object.entries(params ?? {})
      .filter(([, v]) => v !== undefined && v !== null && v !== '')
      .map(([k, v]) => `${encodeURIComponent(k)}=${encodeURIComponent(String(v))}`);

    const qs = entries.length > 0 ? `?${entries.join('&')}` : '';
    return await request<BlobList>(`${base}${qs}`);
  },

  async find(key: string): Promise<Result<BlobMeta>> {
    return await request<BlobMeta>(`${base}/${key}`);
  },

  async download(key: string): Promise<Result<Blob>> {
    return await request<Blob>(
      `${base}/download/${key}`,
      undefined,
      (res) => res.blob(),
    );
  },
};
```

### Step 10: Create domain barrel exports

Create `app/client/documents/index.ts`:

```typescript
export * from './document';
export { DocumentService } from './service';
```

Create `app/client/classifications/index.ts`:

```typescript
export * from './classification';
export { ClassificationService } from './service';
export type { ValidateCommand, UpdateCommand } from './service';
```

Create `app/client/prompts/index.ts`:

```typescript
export * from './prompt';
export { PromptService } from './service';
```

Create `app/client/storage/index.ts`:

```typescript
export * from './blob';
export { StorageService } from './service';
export type { StorageListParams } from './service';
```

### Step 11: Verify view barrel stays clean

`app/client/views/documents/index.ts` should remain as-is — only exporting the view component:

```typescript
export * from './documents-view';
```

No types, services, or state in the view barrel.

### Step 12: Update web-development skill

Update `.claude/skills/web-development/SKILL.md` and relevant references to capture the new conventions:

1. **Domain directory convention**: Types and services in domain directories (`documents/`, `classifications/`), separate from `views/`
2. **Service/state separation**: Services are stateless API wrappers (plain objects with a `base` path, methods return `Result<T>` or `AbortController`). State orchestration (signals, context, `ClassifyProgress`) lives adjacent to views/components.
3. **Component organization**: `components/` for stateful components, `elements/` for pure/stateless elements
4. **View directories**: Only view components in `views/<domain>/`, importing from domain directories

Update `references/services.md` to reflect the stateless service pattern and domain directory import paths (e.g., `@app/documents`).

## Validation Criteria

- [ ] `stream()` accepts `init?: RequestInit` and merges with abort signal
- [ ] `stream()` parses `event:` lines and pairs with `data:` lines via `onEvent(type, data)` callback
- [ ] `stream()` completes when reader reports done (no `[DONE]` sentinel dependency)
- [ ] `ExecutionEvent` in `core/api.ts`, re-exported from `core/index.ts`
- [ ] `Document` and `DocumentStatus` in `documents/document.ts`
- [ ] `Classification` in `classifications/classification.ts`
- [ ] `Prompt`, `PromptStage`, `StageContent`, `CreatePromptCommand`, `UpdatePromptCommand` in `prompts/prompt.ts`
- [ ] `BlobMeta` and `BlobList` in `storage/blob.ts`
- [ ] `ValidateCommand` and `UpdateCommand` in `classifications/service.ts`
- [ ] `DocumentService` — maps all 5 handler methods: `list`, `find`, `search`, `upload`, `delete`
- [ ] `ClassificationService` — maps all 8 handler methods: `list`, `find`, `findByDocument`, `search`, `classify`, `validate`, `update`, `delete`
- [ ] `PromptService` — maps all 11 handler methods: `list`, `stages`, `find`, `instructions`, `spec`, `create`, `update`, `delete`, `search`, `activate`, `deactivate`
- [ ] `StorageService` — maps all 3 handler methods: `list`, `find`, `download`
- [ ] Services use PascalCase names and `base` path constant
- [ ] Services return `Result<T>` for request-response, `AbortController` for streaming
- [ ] No signals, context, or state in services
- [ ] Domain barrels export types and services
- [ ] View barrel only exports view component
- [ ] Web-development skill updated with service/state separation and domain directory conventions
- [ ] `bun run build` compiles without errors
