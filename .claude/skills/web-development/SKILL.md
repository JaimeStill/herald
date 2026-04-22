---
name: web-development
description: >
  REQUIRED for web client development with Lit. Use when creating views,
  modules, elements, services, or styling with CSS layers.
  Triggers: app/client/, LitElement, @customElement, @state, @property,
  design/tokens, "create module", "add view", "add service".
  File patterns: app/**/*.ts, app/**/*.css, app/**/*.go, pkg/web/*.go
---

# Web Development with Lit

## When This Skill Applies

- Creating or modifying web client code in `app/client/`
- Implementing Lit components (views, modules, elements)
- Working with services and component state management
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
| Services | `app/client/domains/<domain>/service.ts` | Stateless API wrappers mirroring Go handlers. Called by views and modules only. |
| Component state | `@state()` decorator | Per-component reactive state — data, filters, progress, errors, UI toggles |
| Props | `@property()` decorator | Data passed from parent to child. Views pass to modules, modules pass to elements. |

Services are stateless — they return `Result<T>` or `AbortController` and forget. Modules call services directly, update their own `@state()` fields, and pass data to child elements via `@property()`. There is no orchestration layer between services and components.

### Three-Tier Component Hierarchy

Each tier has a specific role. Violating the boundaries (e.g., a pure element directly calling an API) creates hidden dependencies that make components harder to test and reuse.

| Tier | Role | Tools | Example |
|------|------|-------|---------|
| View | Route-level composition, coordinate modules, manage view-level state | `@state()`, services, `querySelector` | `hd-documents-view` |
| Module | Self-contained capability unit — owns state, calls services, orchestrates elements | `@state()`, services, events | `hd-document-grid` |
| Element | Pure — props in, events out | `@property`, `CustomEvent`. Imports `lit`, own CSS module, and immutable domain infrastructure (types, constants, formatters). | `hd-document-card` |

**Lego analogy:** Element = brick, Module = car (composed of bricks, functional, self-sufficient), View = scene.

## Reference Guide

Each topic below has a dedicated reference with full code examples and detailed patterns. Read the relevant reference when working in that area.

### Components — [references/components.md](references/components.md)

Three component tiers with complete examples: View components manage view-level `@state()`, compose modules, and coordinate between them via `querySelector` and events. Modules own their data via `@state()`, call services directly, and pass data to elements via `@property()`. Pure elements accept `@property` data and emit `CustomEvent` upward. Every component uses `*.module.css` imports for styles (producing `CSSStyleSheet` directly — no `unsafeCSS()`).

### Services — [references/services.md](references/services.md)

Stateless API wrappers that mirror Go domain handlers. Each domain has a PascalCase service object (`DocumentService`, `ClassificationService`, etc.) with a `base` path constant. Methods return `Result<T>` for request-response and `AbortController` for streaming. No signals, no context, no state.

### Component State — [references/state.md](references/state.md)

`@state()` is the primary state management tool for views and modules. Modules own their data (fetched from services), filters, pagination, and UI state as `@state()` fields. Views manage view-level toggles and coordinate between modules via `querySelector` and events. Data flows down via `@property()`, events flow up via `CustomEvent`.

### CSS — [references/css.md](references/css.md)

Five cascade layers (`tokens, reset, base, theme, app`), design tokens as CSS custom properties using `light-dark()` (penetrate shadow DOM naturally), component styles via `*.module.css` → `CSSStyleSheet`, shared styles via `@styles/*` alias (including `.scroll-y` / `.scroll-x` utilities — the default for every scroll container), global CSS via side-effect import, and app-shell scroll architecture (body never scrolls, views own scroll regions).

### API — [references/api.md](references/api.md)

`Result<T>` discriminated union, `request<T>()` generic fetch wrapper (base `/api`), `stream()` SSE client with `AbortController` cancellation, and `PageResult<T>`/`PageRequest`/`toQueryString` pagination helpers. Located at `app/client/core/api.ts`.

### Router — [references/router.md](references/router.md)

History API router at `app/client/core/router/`. Routes are defined in `app/client/routes.ts` and injected into the `Router` constructor — the router has no knowledge of specific routes. Dynamic `:paramName` segments, catch-all `'*'`, `navigate()` for programmatic routing. Path params are set as HTML attributes on mounted components; query params are read via `queryParams()` and written via `updateQuery()` helpers (no attribute splat). Progressive enhancement with View Transitions API.

### Build — [references/build.md](references/build.md)

Native `Bun.build()` API (no Vite). CSS modules plugin at `app/plugins/css-modules.ts` intercepts `*.module.css` and emits `CSSStyleSheet`. Two-terminal dev workflow: `bun run watch` rebuilds client assets, `air` rebuilds Go on dist/ changes. Output: fixed `app.js` + `app.css` for stable `go:embed`.

### Lifecycles — [references/lifecycles.md](references/lifecycles.md)

Lit lifecycle hooks and when to use each: `connectedCallback` for initial data fetch, `disconnectedCallback` for cleanup, `updated(changed)` for host attribute reflection and property-change reactions, `willUpdate(changed)` for route parameter change handling in views.

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
- **Modules**: `hd-<domain>-<name>` (e.g., `hd-document-grid`)
- **Pure elements**: `hd-<name>` (e.g., `hd-document-card`)
- **Avoid HTMLElement conflicts**: Use `heading` not `title`, `configId` not `id`

### Directory Structure

```
app/client/
├── app.ts                             # entry point — creates Router, imports tiers
├── routes.ts                          # route table (injected into Router)
├── core/                              # framework utilities (no domain knowledge)
│   ├── api.ts                         # request, stream, toQueryString, types
│   ├── index.ts                       # barrel
│   ├── formatting/                    # formatBytes, formatDate
│   └── router/                        # History API router, navigate()
├── design/                            # global design system
│   ├── core/                          # tokens.css, reset.css, theme.css
│   ├── styles/                        # shared component styles (badge, buttons, cards, inputs, labels)
│   ├── app/                           # app-shell styles
│   └── index.css                      # layer declarations + imports
├── domains/                           # data types and service contracts
│   ├── classifications/               # Classification, WorkflowStage, ClassificationService
│   ├── documents/                     # Document, SearchRequest, DocumentService
│   ├── prompts/                       # Prompt, SearchRequest, PromptService
│   └── storage/                       # BlobMeta, StorageService
└── ui/                                # everything that renders (flat per tier)
    ├── elements/                      # pure elements — props in, events out
    │   ├── blob-viewer.ts
    │   ├── classify-progress.ts
    │   ├── confirm-dialog.ts
    │   ├── document-card.ts
    │   ├── markings-list.ts
    │   ├── pagination-controls.ts
    │   ├── prompt-card.ts
    │   └── index.ts                   # barrel
    ├── modules/                       # stateful capability units
    │   ├── classification-panel.ts
    │   ├── document-grid.ts
    │   ├── document-upload.ts
    │   ├── prompt-form.ts
    │   ├── prompt-list.ts
    │   └── index.ts                   # barrel
    └── views/                         # route-level composition
        ├── documents-view.ts
        ├── not-found-view.ts
        ├── prompts-view.ts
        ├── review-view.ts
        └── index.ts                   # barrel
```

Components live directly in their tier directory — no domain subdirectories. The `hd-` prefix already namespaces by domain. Each component has a co-located `*.module.css` file (omitted from tree for brevity).

**Dependency flow:** `design ← core ← domains ← ui`

### Path Aliases

```json
{
  "@core":      "./client/core/index.ts",
  "@core/*":    "./client/core/*",
  "@design/*":  "./client/design/*",
  "@domains/*": "./client/domains/*",
  "@styles/*":  "./client/design/styles/*",
  "@ui/*":      "./client/ui/*"
}
```

### Import Convention

Imports are organized into four groups, separated by blank lines:

1. **Third-party** — `lit`, `lit/decorators.js`, etc.
2. **Cross-package** — path-aliased imports (`@core`, `@domains/*`, etc.)
3. **Relative** — same-package imports (`./document`, `./types`)
4. **Styles** — `@styles/*` first (alphabetically), then relative `*.module.css`

Within each group:
- Infrastructure imports before `type` imports (even if same path)
- Sorted alphabetically by import path, shallower paths first
- PascalCase identifiers before camelCase within the same import

```typescript
import { LitElement, html, nothing } from "lit";
import { customElement, property } from "lit/decorators.js";

import { formatBytes, formatDate } from "@core/formatting";
import type { WorkflowStage } from "@domains/classifications";
import type { Document } from "@domains/documents";

import badgeStyles from "@styles/badge.module.css";
import buttonStyles from "@styles/buttons.module.css";
import styles from "./document-card.module.css";
```

### Custom Events

Event names are simple action verbs describing what happened from the component's perspective. Avoid prefixing with the domain name — the component tag already provides context.

- **Simple verbs**: `select`, `delete`, `save`, `create`, `cancel`
- **Compound verbs** when needed for clarity: `toggle-active`, `upload-complete`, `page-change`
- **No domain prefix**: `select` not `prompt-select`, `delete` not `prompt-deleted`
- **Avoid overwriting native events** that are semantically associated with the component itself (e.g., don't use `input` on a component that wraps an input). A list emitting `select` is fine — the list itself doesn't have a native `select` semantic.

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
- Putting signals or context in service files — services are stateless API wrappers
- Creating state orchestration layers between services and modules — modules call services directly
- Pure elements importing stateful infrastructure — services, context, or router utilities. Elements can import immutable domain infrastructure (types, constants, formatters) but never anything that holds or mutates state.
- Using `height: 100%` in flex containers — use `flex: 1` with `min-height: 0`
- Forgetting `min-height: 0` on flex children that need scroll boundaries
- Declaring `overflow-y: auto` (or `overflow-x: auto`) directly in a component's CSS — use the `.scroll-y` / `.scroll-x` utility from `@styles/scroll.module.css` so the scrollbar gutter and spacing stay consistent across the app
- Using inline `style` attributes — use CSS classes and custom properties
- Accessing `this.id` or `this.title` on components — conflicts with `HTMLElement` built-ins
- Importing `*.module.css` with `?inline` suffix — Herald uses the naming convention, not query params
- Using `margin-left: auto` hacks — let flex container properties (`justify-content`, `gap`) manage layout

### Prefer

- Native HTML elements with CSS classes for simple UI
- `@state()` for all component-owned state — data, filters, pagination, progress, errors, UI toggles
- Modules calling services directly — no orchestration middleman
- Events up (`CustomEvent`), data down (`@property()`) for parent-child communication
- Views coordinating modules via `querySelector` + public methods (e.g., `refresh()`)
- `nothing` from Lit for conditional non-rendering
- FormData extraction over controlled inputs for form handling
- `disconnectedCallback` cleanup for blob URLs and event listeners
- Event delegation at the list level over individual handlers on each item
- Domain types and constants in pure elements — immutable domain knowledge is fine, stateful behavior is not
- Monospace font (`--font-mono`) on interactive elements (buttons, inputs, selects)
- `light-dark()` for color tokens instead of duplicated `@media (prefers-color-scheme)` blocks
- `.scroll-y` / `.scroll-x` from `@styles/scroll.module.css` as the default for every scroll container — layout lives in the component CSS (`flex: 1; min-height: 0;`), scroll behavior lives in the utility
