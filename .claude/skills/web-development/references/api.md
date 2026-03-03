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

// Custom parse (non-JSON response)
const result = await request<Blob>(
  `/storage/download/${key}`,
  undefined,
  (res) => res.blob(),
);
```

## stream()

SSE client for server-sent event streams. Returns an `AbortController` for cancellation. Accepts an optional `RequestInit` for non-GET requests (e.g., POST for classification).

```typescript
function stream(
  path: string,
  options: StreamOptions,
  init?: RequestInit,
): AbortController
```

### StreamOptions

```typescript
interface StreamOptions {
  onEvent: (type: string, data: string) => void;
  onError?: (error: string) => void;
  onComplete?: () => void;
  signal?: AbortSignal;
}
```

### Behavior

- Merges `init` with the abort signal: `fetch(\`${BASE}${path}\`, { ...init, signal })`
- Parses SSE `event:` lines to track the current event type (defaults to `'message'`)
- Pairs each `data:` line with the current event type, then dispatches via `onEvent(type, data)`
- Resets event type to `'message'` after each `data:` dispatch
- Calls `onComplete()` when the reader reports `done`
- Calls `onError()` on fetch failure or non-ok response (ignores `AbortError`)

### Event Parsing

The SSE format uses `event:` and `data:` line pairs:

```
event: node.start
data: {"type":"node.start","timestamp":"...","data":{"node":"classify"}}

event: node.complete
data: {"type":"node.complete","timestamp":"...","data":{"node":"classify"}}

event: complete
data: {"type":"complete","timestamp":"...","data":{}}
```

The parser tracks the current event type from `event:` lines and pairs it with the next `data:` line:

```typescript
let currentEvent = 'message';

for (const line of lines) {
  if (line.startsWith('event: ')) {
    currentEvent = line.slice(7).trim();
  } else if (line.startsWith('data: ')) {
    const data = line.slice(6).trim();
    options.onEvent(currentEvent, data);
    currentEvent = 'message';
  }
}
```

### Usage Example

```typescript
const controller = ClassificationService.classify(documentId, {
  onEvent(type, data) {
    const event = JSON.parse(data);
    switch (type) {
      case 'node.start':
        // Update progress UI with event.data.node
        break;
      case 'node.complete':
        if (event.data.error) {
          // Handle node-level error
        }
        break;
      case 'complete':
        // Classification finished
        break;
      case 'error':
        // Workflow-level error
        break;
    }
  },
  onError(err) {
    // Fetch/network failure
  },
  onComplete() {
    // Stream ended (reader done)
  },
});

// Cancel if needed
controller.abort();
```

## ExecutionEvent

Typed structure for SSE events from the classification workflow:

```typescript
interface ExecutionEvent {
  type: string;
  timestamp: string;
  data: Record<string, unknown>;
}
```

Lives in `core/api.ts` as SSE streaming infrastructure. The `data` field in `onEvent` callbacks is the raw JSON string — parse it to get the `ExecutionEvent` shape.

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
