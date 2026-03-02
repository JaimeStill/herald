---
name: web-development
description: >
  REQUIRED for web client development with Lit. Use when creating views,
  components, elements, services, or styling with CSS layers.
  Triggers: app/client/, LitElement, @customElement, @provide, @consume,
  SignalWatcher, design/tokens, "create component", "add view", "add service".
  File patterns: app/**/*.ts, app/**/*.css, app/**/*.go, pkg/web/*.go
---

# Web Development with Lit

## When This Skill Applies

- Creating or modifying web client code in `app/client/`
- Implementing Lit components (views, stateful components, elements)
- Working with services and context-based dependency injection
- Styling with CSS cascade layers and design tokens
- Integrating Go server with Lit client (`app/app.go`, `app/server/`)
- Build system work (`app/scripts/`, `app/plugins/`)

## Architecture Overview

### Hard Boundary Principle

**Go owns data and routing, Lit owns presentation entirely.**

- Go serves a single HTML shell for all `/app/*` routes
- Client-side router handles view mounting
- No server-side view awareness for client routes

### Three-Tier Component Hierarchy

Each tier has a specific role. Violating the boundaries (e.g., a pure element directly calling an API) creates hidden dependencies that make components harder to test and reuse.

| Tier | Role | Tools | Example |
|------|------|-------|---------|
| View | Provide services, route-level | `@provide`, `SignalWatcher` | `hd-documents-view` |
| Stateful Component | Consume services, coordinate UI | `@consume`, event handlers | `hd-document-list` |
| Pure Element | Props in, events out | `@property`, `CustomEvent` | `hd-document-card` |

## Reference Guide

Each topic below has a dedicated reference with full code examples and detailed patterns. Read the relevant reference when working in that area.

### Components — [references/components.md](references/components.md)

Three component tiers with complete examples: View components create and `@provide` services via `SignalWatcher(LitElement)`. Stateful components `@consume` services and coordinate UI. Pure elements accept `@property` data and emit `CustomEvent` upward. Every component uses `*.module.css` imports for styles (producing `CSSStyleSheet` directly — no `unsafeCSS()`).

### Services — [references/services.md](references/services.md)

Each domain has a single `service.ts` exporting a context, interface, and factory function. State lives in `Signal.State` signals. Views create services via factory and `@provide` them; descendants `@consume`. The `SignalWatcher` mixin drives re-renders when signals change.

### CSS — [references/css.md](references/css.md)

Three cascade layers (`tokens, reset, theme`), design tokens as CSS custom properties (penetrate shadow DOM naturally), component styles via `*.module.css` → `CSSStyleSheet`, global CSS via side-effect import, and app-shell scroll architecture (body never scrolls, views own scroll regions).

### API — [references/api.md](references/api.md)

`Result<T>` discriminated union, `request<T>()` generic fetch wrapper (base `/api`), `stream()` SSE client with `AbortController` cancellation, and `PageResult<T>`/`PageRequest`/`toQueryString` pagination helpers. Located at `app/client/core/api.ts`.

### Router — [references/router.md](references/router.md)

History API router at `app/client/router/`. Route definitions map path patterns to component tag names. Dynamic `:paramName` segments, catch-all `'*'`, `navigate()` for programmatic routing. Router sets path/query params as HTML attributes on mounted components.

### Build — [references/build.md](references/build.md)

Native `Bun.build()` API (no Vite). CSS modules plugin at `app/plugins/css-modules.ts` intercepts `*.module.css` and emits `CSSStyleSheet`. Two-terminal dev workflow: `bun run watch` rebuilds client assets, `air` rebuilds Go on dist/ changes. Output: fixed `app.js` + `app.css` for stable `go:embed`.

### Go Integration — [references/go-integration.md](references/go-integration.md)

Single shell pattern: `app/app.go` embeds `dist/*`, `server/layouts/*`, `server/views/*`. Catch-all `/{path...}` route serves the HTML shell. Template variables: `{{ .BasePath }}`, `{{ .Title }}`, `{{ .Bundle }}`. `<base href>` tag enables client-side router path resolution.

## Template Patterns

These patterns recur across all component tiers and are worth keeping top of mind.

**Render methods** — extract complex template logic into private `renderXxx()` methods. Use `nothing` from Lit (not empty string) for conditional non-rendering.

**Form handling** — extract values via `FormData` on submit rather than tracking controlled inputs. Cast with `data.get('name') as string`.

**Host attribute reflection** — reflect `@state()` to host attributes via `updated()` + `toggleAttribute()` so CSS can drive layout changes without JavaScript.

**Object URL lifecycle** — revoke blob URLs in `disconnectedCallback` to prevent memory leaks. Use a `Map<File, string>` cache pattern.

See [references/components.md](references/components.md) for full code examples of each pattern.

## Naming Conventions

### Components

- **Prefix**: `hd-` (Herald)
- **Views**: `hd-<domain>-view` (e.g., `hd-documents-view`)
- **Stateful components**: `hd-<domain>-<name>` (e.g., `hd-document-list`)
- **Pure elements**: `hd-<name>` (e.g., `hd-document-card`)
- **Avoid HTMLElement conflicts**: Use `heading` not `title`, `configId` not `id`

### File Structure

Each view lives in its own subdirectory with a barrel export:

```
views/documents/
├── index.ts                     # barrel export
├── documents-view.ts            # @customElement('hd-documents-view')
├── documents-view.module.css    # component styles
├── service.ts                   # domain service + context + factory
└── document-card.ts             # pure element (co-located if small)
```

Shared components go in `app/client/components/` with the same pattern.

### Path Alias

`@app/*` resolves to `app/client/*` (configured in `tsconfig.json`):

```typescript
import { request } from '@app/core';
import { navigate } from '@app/router';
```

### HTMLElementTagNameMap

Every component declares its tag in the global interface for type safety:

```typescript
declare global {
  interface HTMLElementTagNameMap {
    'hd-documents-view': DocumentsView;
  }
}
```

## Anti-Patterns

### Avoid

- Creating custom elements for native HTML (buttons, inputs, badges) — use CSS classes
- Using `unsafeCSS()` — Herald's `*.module.css` plugin produces `CSSStyleSheet` directly
- Storing service references in `@state()` — use `@consume` for context injection
- Skipping `SignalWatcher` mixin when consuming signal-based services (reactivity won't work)
- Using `height: 100%` in flex containers — use `flex: 1` with `min-height: 0`
- Forgetting `min-height: 0` on flex children that need scroll boundaries
- Using inline `style` attributes — use CSS classes and custom properties
- Accessing `this.id` or `this.title` on components — conflicts with `HTMLElement` built-ins
- Importing `*.module.css` with `?inline` suffix — Herald uses the naming convention, not query params

### Prefer

- Native HTML elements with CSS classes for simple UI
- `@provide`/`@consume` over prop drilling through intermediate components
- `nothing` from Lit for conditional non-rendering
- FormData extraction over controlled inputs for form handling
- `disconnectedCallback` cleanup for blob URLs and event listeners
- Event delegation at the list level over individual handlers on each item
