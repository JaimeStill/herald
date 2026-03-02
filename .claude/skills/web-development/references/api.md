# API Layer

Located at `app/client/core/api.ts`. All exports are re-exported from `app/client/core/index.ts`.

## Result Type

Discriminated union for consistent error handling across all API calls:

```typescript
type Result<T> =
  | { ok: true; data: T }
  | { ok: false; error: string };
```

Check `result.ok` before accessing `result.data`. TypeScript narrows the type automatically.

## request()

Generic fetch wrapper with base endpoint `/api`:

```typescript
async function request<T>(
  path: string,
  init?: RequestInit,
  parse?: (res: Response) => Promise<T>
): Promise<Result<T>>
```

Behavior:
- Prepends `/api` to the path
- Non-2xx responses extract the body as the error message
- 204 responses return `undefined` as data
- Custom `parse` hook for non-JSON responses (default: `res.json()`)

### Usage Examples

```typescript
// GET (default)
const result = await request<Document[]>('/documents');

// POST with JSON body
const result = await request<Document>('/documents', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify(command),
});

// POST with FormData (file upload — no Content-Type header, browser sets boundary)
const form = new FormData();
form.append('file', file);
const result = await request<Document>('/documents', {
  method: 'POST',
  body: form,
});

// DELETE
const result = await request<void>(`/documents/${id}`, { method: 'DELETE' });
```

## stream()

SSE client for classification progress. Returns an `AbortController` for cancellation.

```typescript
function stream(
  path: string,
  callbacks: {
    onMessage: (data: string) => void;
    onError?: (err: string) => void;
    onComplete?: () => void;
  }
): AbortController
```

Behavior:
- Parses SSE `data:` lines
- Recognizes `[DONE]` as the completion signal
- Calls `onComplete` when the stream ends normally
- Calls `onError` on fetch failure or stream errors

### Usage Example

```typescript
const controller = stream(`/classifications/${documentId}/stream`, {
  onMessage(data) {
    const event = JSON.parse(data);
    // Update progress UI
  },
  onError(err) {
    console.error('Stream error:', err);
  },
  onComplete() {
    // Classification finished, refresh data
  },
});

// Cancel if needed
controller.abort();
```

## Pagination Helpers

```typescript
interface PageResult<T> {
  data: T[];
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
}

interface PageRequest {
  page?: number;
  page_size?: number;
  search?: string;
  sort?: string;
}

function toQueryString(params: PageRequest): string
```

### Usage

```typescript
const params: PageRequest = { page: 1, page_size: 20, search: 'classified' };
const qs = toQueryString(params); // "?page=1&page_size=20&search=classified"
const result = await request<PageResult<Document>>(`/documents${qs}`);
```
