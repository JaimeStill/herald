# Router

History API router at `app/client/core/router/`. Routes are defined externally in `app/client/routes.ts` and injected into the `Router` constructor — the router has no knowledge of specific routes.

## Route Definitions

```typescript
// app/client/routes.ts
import type { RouteConfig } from "@core/router";

export const routes: Record<string, RouteConfig> = {
  '': { component: 'hd-documents-view', title: 'Documents' },
  'prompts': { component: 'hd-prompts-view', title: 'Prompts' },
  'review/:documentId': { component: 'hd-review-view', title: 'Review' },
  '*': { component: 'hd-not-found-view', title: 'Not Found' },
};
```

- Empty string `''` = root route
- `:paramName` syntax for dynamic segments
- `'*'` = catch-all 404

### Adding a New Route

1. Add entry to `routes` in `app/client/routes.ts`
2. Create the view component in `ui/views/`
3. Export from `ui/views/index.ts` barrel

## Initialization

```typescript
// app/client/app.ts
import { Router } from "@core/router";
import { routes } from "./routes";

const router = new Router("app-content", routes);
router.start();
```

## Navigation

```typescript
import { navigate } from '@core/router';

// Programmatic navigation
navigate('prompts');
navigate(`review/${documentId}`);

// Template links (router intercepts anchor clicks)
html`<a href="prompts">Prompts</a>`
```

## Parameter Passing

The router sets path params and query params as HTML attributes on the mounted component. Components receive them via `@property()`:

```typescript
// Route: 'review/:documentId'
// URL: /app/review/abc-123?tab=markings

@property({ type: String }) documentId?: string;  // "abc-123"
```

Query params are also set as attributes on the component.

## Route Types

```typescript
// core/router/types.ts
interface RouteConfig {
  component: string;      // Custom element tag name
  title: string;          // Page title
}

interface RouteMatch {
  config: RouteConfig;
  params: Record<string, string>;   // Path params
  query: Record<string, string>;    // Query string params
}
```

## How It Works

1. Router receives route table via constructor — no internal route knowledge
2. Reads `<base href>` from the HTML shell for basePath
3. On navigation, strips basePath and matches against route patterns
4. Exact match first, then segment-by-segment pattern matching
5. Creates the custom element by tag name, sets params as attributes
6. Replaces container (`#app-content`) innerHTML with the new element
7. Updates `document.title` and pushes to history
