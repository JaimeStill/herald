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

### Services and State

| Concern | Location | Purpose |
|---------|----------|---------|
| Services | `app/client/<domain>/service.ts` | Stateless API wrappers mirroring Go handlers |
| Shared state | View/component class fields | `Signal.State` signals shared via `@lit/context` |
| Local state | `@state()` decorator | Per-component reactive state (progress, errors, UI toggles) |

Services are stateless — they return `Result<T>` or `AbortController` and forget. Components call services directly, update their own state (signals or `@state()`), and share reactive data with descendants via `@provide`/`@consume`. There is no orchestration layer between services and components.

### Three-Tier Component Hierarchy

Each tier has a specific role. Violating the boundaries (e.g., a pure element directly calling an API) creates hidden dependencies that make components harder to test and reuse.

| Tier | Role | Tools | Example |
|------|------|-------|---------|
| View | Call services, provide shared signals, route-level | `@provide`, `SignalWatcher`, services | `hd-documents-view` |
| Stateful Component | Consume shared state, call services for own concerns | `@consume`, `@state()`, services, events | `hd-document-list` |
| Pure Element | Props in, events out | `@property`, `CustomEvent` | `hd-document-card` |

## Reference Guide

Each topic below has a dedicated reference with full code examples and detailed patterns. Read the relevant reference when working in that area.

### Components — [references/components.md](references/components.md)

Three component tiers with complete examples: View components call services, manage `Signal.State` signals, and `@provide` shared data via `SignalWatcher(LitElement)`. Stateful components `@consume` shared state and call services for their own concerns. Pure elements accept `@property` data and emit `CustomEvent` upward. Every component uses `*.module.css` imports for styles (producing `CSSStyleSheet` directly — no `unsafeCSS()`).

### Services — [references/services.md](references/services.md)

Stateless API wrappers that mirror Go domain handlers. Each domain has a PascalCase service object (`DocumentService`, `ClassificationService`, etc.) with a `base` path constant. Methods return `Result<T>` for request-response and `AbortController` for streaming. No signals, no context, no state.

### Shared Reactive State — [references/state.md](references/state.md)

`Signal.State` signals shared across component subtrees via `@lit/context`. Views `@provide` signals as class fields; descendants `@consume` them. Components call services directly and update signals themselves — no factory functions or orchestration layer. `@state()` for local concerns (progress, errors, UI toggles). `SignalWatcher` mixin drives re-renders.

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

Domain infrastructure (types, services) lives in domain directories. Views, components, and elements each have their own top-level directories.

```
app/client/
├── core/                            # API layer (request, stream, types)
├── documents/                       # domain: types + service
│   ├── document.ts                  # Document, DocumentStatus
│   ├── service.ts                   # DocumentService (stateless)
│   └── index.ts                     # barrel
├── classifications/                 # domain: types + service
│   ├── classification.ts            # Classification
│   ├── service.ts                   # ClassificationService (stateless)
│   └── index.ts                     # barrel
├── prompts/                         # domain: types + service
├── storage/                         # domain: types + service
├── views/
│   └── documents/                   # view: route-level component
│       ├── index.ts                 # barrel (view component only)
│       ├── documents-view.ts        # @customElement('hd-documents-view')
│       └── documents-view.module.css
├── components/                      # stateful components (@consume)
├── elements/                        # pure elements (props/events)
├── router/
└── design/
```

**Domain directories** export types and stateless services. **View directories** export view components. **Components** and **elements** are shared across views.

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
- Skipping `SignalWatcher` mixin when consuming signal-based state (reactivity won't work)
- Putting signals or context in service files — services are stateless API wrappers
- Creating state orchestration layers between services and components — components call services directly
- Pure elements calling services — only views and stateful components should import services
- Using `height: 100%` in flex containers — use `flex: 1` with `min-height: 0`
- Forgetting `min-height: 0` on flex children that need scroll boundaries
- Using inline `style` attributes — use CSS classes and custom properties
- Accessing `this.id` or `this.title` on components — conflicts with `HTMLElement` built-ins
- Importing `*.module.css` with `?inline` suffix — Herald uses the naming convention, not query params

### Prefer

- Native HTML elements with CSS classes for simple UI
- `@provide`/`@consume` over prop drilling through intermediate components
- `@state()` for local component concerns (progress, errors, UI toggles)
- `Signal.State` via context only for data shared across multiple descendants
- Components calling services directly — no orchestration middleman
- Events up (`CustomEvent`), data down (`@property`, context) for parent-child communication
- `nothing` from Lit for conditional non-rendering
- FormData extraction over controlled inputs for form handling
- `disconnectedCallback` cleanup for blob URLs and event listeners
- Event delegation at the list level over individual handlers on each item
