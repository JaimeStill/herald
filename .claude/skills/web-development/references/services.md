# Services

Stateless API wrappers that mirror Go domain handlers. Each domain has a PascalCase service object with a `base` path constant. Methods return `Result<T>` for request-response and `AbortController` for streaming. No signals, no context, no state.

## SearchRequest Convention

Each domain defines its own `SearchRequest` interface combining pagination fields with domain-specific filters. Both `list()` (GET + query params) and `search()` (POST + JSON body) accept it. The type mirrors the Go handler's `SearchRequest` struct (embedded `PageRequest` + `Filters`), which serializes to flat JSON.

```typescript
// documents/document.ts — domain-specific SearchRequest
export interface SearchRequest {
  page?: number;
  page_size?: number;
  search?: string;
  sort?: string;
  status?: string;           // domain filter
  classification?: string;   // domain filter
  confidence?: string;       // domain filter
}
```

`toQueryString()` in `@core` is generic (`<T extends object>`) and serializes any `SearchRequest` variant to query params.

## Service Pattern

```typescript
// documents/service.ts
import { request, toQueryString, type Result, type PageResult } from '@core';
import type { Document, SearchRequest } from './document';

const base = '/documents';

export const DocumentService = {
  async list(params?: SearchRequest): Promise<Result<PageResult<Document>>> {
    return await request<PageResult<Document>>(
      `${base}${params ? toQueryString(params) : ''}`
    );
  },

  async search(body: SearchRequest): Promise<Result<PageResult<Document>>> {
    return await request<PageResult<Document>>(`${base}/search`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    });
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
import { stream, type StreamOptions } from '@core';

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
- **Domain directories**: Services live in `app/client/domains/<domain>/service.ts`, not in view directories
- **Barrel exports**: `export { DocumentService } from './service'` — named export, not `export *`

## Available Services

| Service | Domain | Import |
|---------|--------|--------|
| `DocumentService` | `documents/` | `import { DocumentService } from '@domains/documents'` |
| `ClassificationService` | `classifications/` | `import { ClassificationService } from '@domains/classifications'` |
| `PromptService` | `prompts/` | `import { PromptService } from '@domains/prompts'` |
| `StorageService` | `storage/` | `import { StorageService } from '@domains/storage'` |
