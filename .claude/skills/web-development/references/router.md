# Router

History API router at `app/client/router/`. Handles client-side navigation within the SPA.

## Route Definitions

```typescript
// router/routes.ts
const routes: Record<string, RouteConfig> = {
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

1. Add entry to `routes` in `router/routes.ts`
2. Create the view component in `views/<domain>/`
3. Import the view in `views/index.ts` barrel

## Navigation

```typescript
import { navigate } from '@app/router';

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
// router/types.ts
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

1. Router reads `<base href>` from the HTML shell for basePath
2. On navigation, strips basePath and matches against route patterns
3. Exact match first, then segment-by-segment pattern matching
4. Creates the custom element by tag name, sets params as attributes
5. Replaces container (`#app-content`) innerHTML with the new element
6. Updates `document.title` and pushes to history
