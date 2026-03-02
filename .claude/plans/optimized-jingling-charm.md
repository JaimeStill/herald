# Issue #63 — Web Client Build System, Design, and Client Application

## Context

Phase 3 adds a Lit 3.x web client to Herald. Issue #63 establishes the complete client-side foundation under `app/`: Bun build pipeline with CSS module plugin, design system with cascade layers, History API router, core API layer, and placeholder views for all routes. This is purely client-side — Go integration (app.go, templates, module mounting, Air config) is issue #64.

Root directory is `app/` (not `web/app/`) with Go `package app` — clean, no collision with `pkg/web`, conventional naming.

Patterns adapted from agent-lab (`~/code/agent-lab/web/app/`) with key differences: Bun replaces Vite, `{ type: 'css' }` import attributes replace `?inline`, `hd-` prefix replaces `lab-`, Herald-specific routes/branding.

## Implementation Steps

### Step 1: Package and TypeScript Setup

**`app/package.json`** — Bun project with `lit`, `@lit/context`, `@lit-labs/signals`. Scripts: `build` and `watch`.

**`app/tsconfig.json`** — ES2024 target, `@app` path alias to `./client/*`, bundler module resolution, decorator support (`experimentalDecorators`, `useDefineForClassFields: false`).

**`app/client/css.d.ts`** — Declares `*.css` modules returning `CSSStyleSheet` (for `{ type: 'css' }` imports).

### Step 2: CSS Module Plugin

**`app/plugins/css-modules.ts`** — Bun plugin using `onResolve`/`onLoad` hooks:
- `onResolve`: checks `args.with?.type === 'css'` — matched imports get namespace `css-module`; unmatched return undefined (fall through to Bun's default CSS extraction)
- `onLoad`: reads CSS file, escapes backticks/`${`, emits JS module constructing `CSSStyleSheet` via `replaceSync()`
- Result: component CSS inlined in app.js as CSSStyleSheet modules, global CSS extracted to app.css

### Step 3: Build and Watch Scripts

**`app/scripts/build.ts`** — `Bun.build()` with entry `client/app.ts`, plugin, output to `dist/`. Fixed filenames (`app.js`, `app.css`) for stable `go:embed`.

**`app/scripts/watch.ts`** — `fs.watch` on `client/` for `*.ts` and `*.css` changes, triggers rebuild. Debounced to avoid rapid successive builds.

### Step 4: Design System

CSS cascade layers adapted from agent-lab:

- **`client/design/index.css`** — `@layer tokens, reset, theme;` + `@import` chain
- **`client/design/core/tokens.css`** — CSS custom properties: colors (dark/light via `prefers-color-scheme`), spacing scale (`--space-1` to `--space-16`), typography scale (`--text-xs` to `--text-4xl`), font stacks
- **`client/design/core/reset.css`** — Modern CSS reset (box-sizing, margin, min-height)
- **`client/design/core/theme.css`** — Body font-family, background, color from tokens
- **`client/design/app/app.css`** — App shell layout: flex column body, fixed header with brand "Herald" + nav (Documents, Prompts), flex-1 main content area
- **`client/design/app/elements.css`** — Base element styles for Shadow DOM components (placeholder, expanded as components are built)

### Step 5: Client-Side Router

Ported from agent-lab's History API router:

- **`client/router/types.ts`** — `RouteConfig` (component, title), `RouteMatch` (config, params, query)
- **`client/router/routes.ts`** — Route map: `''` → `hd-documents-view`, `'prompts'` → `hd-prompts-view`, `'documents/:documentId/review'` → `hd-review-view`, `'*'` → `hd-not-found-view`
- **`client/router/router.ts`** — Router class with `<base href>` awareness, `:param` matching, query string parsing, dynamic component mounting, popstate handling. Title suffix: "Herald"
- **`client/router/index.ts`** — Public re-exports

### Step 6: Core API Layer

**`client/core/api.ts`** — Simplified from agent-lab:
- `Result<T>` discriminated union (`{ ok: true, data: T } | { ok: false, error: string }`)
- `request<T>(path, init?, parse?)` — fetch wrapper with `/api` base, JSON parsing, error handling, 204 support
- `stream(path, options)` — SSE via fetch + ReadableStream, returns `AbortController`. Herald-specific `StreamOptions`: `onMessage(data)`, `onError(error)`, `onComplete()`, optional `signal`
- `PageResult<T>`, `PageRequest`, `toQueryString()` helper

**`client/core/index.ts`** — Public re-exports.

### Step 7: Placeholder Views

Four views, each with `.ts` + `.css`:

- **`client/views/not-found-view.ts`** + `.css` — 404 page showing missed path
- **`client/views/documents-view.ts`** + `.css` — "Documents" placeholder
- **`client/views/prompts-view.ts`** + `.css` — "Prompts" placeholder
- **`client/views/review-view.ts`** + `.css` — "Review" placeholder (receives `documentId` attribute from router params)

Pattern: `@customElement('hd-*')`, `import styles from './x.css' with { type: 'css' }`, `static styles = styles`, `HTMLElementTagNameMap` declaration.

### Step 8: Entry Point

**`client/app.ts`**:
1. `import './design/index.css'` — side-effect for global CSS extraction
2. Import all view components (side-effect registrations)
3. `import { Router } from '@app/router'`
4. Create Router with container ID `'app-content'`, call `start()`

### Step 9: Gitignore and Initial Build

- Add `app/node_modules/` and `app/dist/` to `.gitignore`
- Run `bun install` then `bun run build` to verify pipeline produces `dist/app.js` + `dist/app.css`

## Files Created

```
app/
├── package.json
├── tsconfig.json
├── plugins/
│   └── css-modules.ts
├── scripts/
│   ├── build.ts
│   └── watch.ts
└── client/
    ├── app.ts
    ├── css.d.ts
    ├── core/
    │   ├── api.ts
    │   └── index.ts
    ├── design/
    │   ├── index.css
    │   ├── core/
    │   │   ├── tokens.css
    │   │   ├── reset.css
    │   │   └── theme.css
    │   └── app/
    │       ├── app.css
    │       └── elements.css
    ├── router/
    │   ├── types.ts
    │   ├── routes.ts
    │   ├── router.ts
    │   └── index.ts
    └── views/
        ├── not-found-view.ts
        ├── not-found-view.css
        ├── documents-view.ts
        ├── documents-view.css
        ├── prompts-view.ts
        ├── prompts-view.css
        ├── review-view.ts
        └── review-view.css
```

## Validation Criteria

- [ ] `bun install` succeeds with lit, @lit/context, @lit-labs/signals
- [ ] `bun run build` produces `dist/app.js` and `dist/app.css`
- [ ] CSS module plugin correctly discriminates `{ type: 'css' }` from side-effect imports
- [ ] Component CSS inlined in `app.js` as CSSStyleSheet modules (not in `app.css`)
- [ ] Global design CSS extracted to `dist/app.css`
- [ ] `bun run watch` rebuilds on `client/**/*.{ts,css}` changes
- [ ] Router navigates between placeholder views (fully testable after #64)
- [ ] `Result<T>` API layer compiles with correct types
- [ ] All 4 placeholder views render with `hd-` prefix
