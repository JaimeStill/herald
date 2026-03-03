# Services

Stateless API wrappers that mirror Go domain handlers. Each domain has a PascalCase service object with a `base` path constant. Methods return `Result<T>` for request-response and `AbortController` for streaming. No signals, no context, no state.

## Service Pattern

```typescript
// documents/service.ts
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

## Streaming Endpoints

SSE endpoints return `AbortController`. The caller provides `StreamOptions` callbacks, giving the state layer full control over event processing.

```typescript
// classifications/service.ts
import { stream, type StreamOptions } from '@app/core';

const base = '/classifications';

export const ClassificationService = {
  classify(documentId: string, options: StreamOptions): AbortController {
    return stream(`${base}/${documentId}`, options, { method: 'POST' });
  },
  // ...other methods
};
```

## Conventions

- **PascalCase names**: `DocumentService`, `ClassificationService`, `PromptService`, `StorageService`
- **`base` constant**: Each service captures its API prefix once — methods use `${base}/...`
- **Mirror Go handlers**: Every handler method has a corresponding service method with matching semantics
- **`Result<T>` returns**: Request-response methods return `Result<T>` directly from `request()`
- **`AbortController` returns**: Streaming methods return the controller from `stream()`
- **No state**: Services never import `Signal`, `createContext`, or `@lit/context`
- **Domain directories**: Services live in `app/client/<domain>/service.ts`, not in view directories
- **Barrel exports**: `export { DocumentService } from './service'` — named export, not `export *`

## Available Services

| Service | Domain | Import |
|---------|--------|--------|
| `DocumentService` | `documents/` | `import { DocumentService } from '@app/documents'` |
| `ClassificationService` | `classifications/` | `import { ClassificationService } from '@app/classifications'` |
| `PromptService` | `prompts/` | `import { PromptService } from '@app/prompts'` |
| `StorageService` | `storage/` | `import { StorageService } from '@app/storage'` |
