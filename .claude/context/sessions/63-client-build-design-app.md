# 63 - Web Client Build System, Design, and Client Application

## Summary

Established the complete client-side foundation for Herald's Lit 3.x web client under `app/`. Implemented a Bun build pipeline with a CSS module plugin that discriminates `*.module.css` component styles from plain CSS global styles, a design system using CSS cascade layers with dark/light theme support, a History API router adapted from agent-lab, a core API layer with `Result<T>` error handling and SSE streaming, and placeholder views for all four routes.

## Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Root directory | `app/` (not `web/app/`) | Herald has a single web client — no need for the multi-client nesting agent-lab uses. `package app` avoids naming collision with `pkg/web`. |
| CSS module discrimination | `*.module.css` naming convention | Bun 1.3.10 does not expose import attributes to plugin hooks ([oven-sh/bun#7293](https://github.com/oven-sh/bun/issues/7293)). File naming is fail-safe — the file itself declares intent, preventing accidental misuse. |
| Review route | `review/:documentId` | Flat top-level route instead of nested `documents/:documentId/review`. |
| View organization | Subdirectories with barrel exports | Each view in its own directory (`views/documents/`, `views/not-found/`, etc.) with `index.ts` re-exports. Clean separation as views grow. |
| Watch script filtering | `.ts` and `.css` only | No HTML in `client/` — shell HTML is a Go template (issue #64), not part of the Bun pipeline. |

## Files Created

- `app/package.json` — Bun project with lit, @lit/context, @lit-labs/signals, @types/bun
- `app/tsconfig.json` — ES2024, @app path alias, decorator support
- `app/bun.lock` — Dependency lock file
- `app/plugins/css-modules.ts` — Bun plugin: `*.module.css` → CSSStyleSheet modules
- `app/scripts/build.ts` — Bun.build() with plugin, outputs dist/app.js + dist/app.css
- `app/scripts/watch.ts` — Debounced file watcher for client/ changes
- `app/client/css.d.ts` — TypeScript declaration for `*.module.css` imports
- `app/client/app.ts` — Entry point: design import, view registration, router start
- `app/client/core/api.ts` — Result<T>, request(), stream(), PageResult, toQueryString()
- `app/client/core/index.ts` — Barrel exports
- `app/client/design/index.css` — @layer declarations + @import chain
- `app/client/design/core/tokens.css` — CSS custom properties (colors, spacing, typography)
- `app/client/design/core/reset.css` — Modern CSS reset
- `app/client/design/core/theme.css` — Body styling from tokens
- `app/client/design/app/app.css` — App shell layout (header, nav, content area)
- `app/client/design/app/elements.css` — Placeholder for Shadow DOM base styles
- `app/client/router/types.ts` — RouteConfig, RouteMatch
- `app/client/router/routes.ts` — Route map (documents, prompts, review, not-found)
- `app/client/router/router.ts` — History API router with base href, params, query
- `app/client/router/index.ts` — Barrel exports
- `app/client/views/index.ts` — Top-level barrel for all views
- `app/client/views/documents/{index.ts, documents-view.ts, documents-view.module.css}
- `app/client/views/not-found/{index.ts, not-found-view.ts, not-found-view.module.css}
- `app/client/views/prompts/{index.ts, prompts-view.ts, prompts-view.module.css}
- `app/client/views/review/{index.ts, review-view.ts, review-view.module.css}

## Patterns Established

- **CSS module convention**: `*.module.css` → `CSSStyleSheet` via Bun plugin; plain `*.css` → global extraction to `dist/app.css`
- **Component pattern**: `@customElement('hd-*')`, `import styles from './x.module.css'`, `static styles = styles`, `HTMLElementTagNameMap` declaration
- **View organization**: Each view in its own subdirectory with barrel `index.ts`
- **Entry point**: Side-effect imports for design CSS and view registration, then router instantiation
- **Build output**: Fixed names `app.js`/`app.css` in `dist/` for stable `go:embed` (issue #64)

## Validation Results

- `bun run build` produces `dist/app.js` (64KB) + `dist/app.css` (3.5KB)
- Component CSS inlined in app.js as CSSStyleSheet (8 refs, 5 replaceSync calls)
- Global design CSS extracted to app.css (tokens, reset, theme, app layout)
- `go vet ./...` clean
- All 19 existing Go test suites pass
- `bun run watch` rebuilds on client/ changes
