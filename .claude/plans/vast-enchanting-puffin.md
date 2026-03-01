# Objective Planning — #57 Web Client Foundation and Build System

## Context

This is the first Phase 3 objective. It establishes the complete web client infrastructure that all subsequent web objectives (#58–#61) depend on. The objective covers Go-side template/asset serving, a Lit 3.x SPA with native Bun builds, client-side routing, a design system, a core API layer, Air hot reload, and a web-development skill.

**CSS approach update**: The original issue specifies `{ type: 'text' }` + `unsafeCSS()`. A refined approach using `{ type: 'css' }` with a Bun build plugin that emits native `CSSStyleSheet` objects was developed separately. This is strictly better — Lit 3+ accepts `CSSStyleSheet` directly in `static styles`, no `unsafeCSS()` wrapper needed, and it mirrors native CSS Module Scripts semantics. The sub-issues below reflect this updated approach.

## Transition Closeout

`_project/objective.md` reads "No active objective" — no transition closeout needed.

## Sub-Issue Decomposition

### Sub-issue 1: `pkg/web/` Template and Static File Infrastructure

**Labels**: `feature`, `infrastructure`
**Milestone**: `v0.3.0 - Web Client`

Port TemplateSet, Router, and static file serving from `~/code/agent-lab/pkg/web/` to `herald/pkg/web/`.

**Scope**:
- `pkg/web/views.go` — `TemplateSet` (pre-parsed template cache), `ViewDef`, `ViewData`, `PageHandler()`, `ErrorHandler()`
- `pkg/web/static.go` — `DistServer()`, `PublicFile()`, `PublicFileRoutes()`, `ServeEmbeddedFile()`
- `pkg/web/router.go` — Router wrapper with fallback handler support
- Remove existing `pkg/web/doc.go` placeholder

**Reference**: `~/code/agent-lab/pkg/web/` (views.go, static.go, router.go)

**Dependencies**: None

---

### Sub-issue 2: Web Client Build System, Design, and Client Application

**Labels**: `feature`, `infrastructure`
**Milestone**: `v0.3.0 - Web Client`

Establish the complete client-side foundation: Bun build pipeline with CSS module plugin, design system, History API router, core API layer, and placeholder views.

**Scope**:

*Build system*:
- `web/app/package.json` — Bun project with `lit`, `@lit/context`, `@lit-labs/signals`
- `web/app/tsconfig.json` — TypeScript config with `@app` path alias
- `web/app/scripts/build-plugin.ts` — `litCSSModulePlugin` discriminating `{ type: 'css' }` imports (emits `CSSStyleSheet` modules) from side-effect imports (Bun default extraction)
- `web/app/scripts/build.ts` — `Bun.build()` with plugin, outputting `dist/app.js` + `dist/app.css`
- `web/app/scripts/watch.ts` — File watcher triggering rebuild on `client/**/*.{ts,css}` changes

*TypeScript declarations*:
- `web/app/client/css.d.ts` — Declares `*.css` module returning `CSSStyleSheet`

*Design system*:
- `web/app/client/design/index.css` — Cascade layer declarations + imports
- `web/app/client/design/core/tokens.css` — CSS custom properties (colors, spacing, typography)
- `web/app/client/design/core/reset.css` — CSS reset/normalize
- `web/app/client/design/core/theme.css` — Color scheme, theming
- `web/app/client/design/app/app.css` — App layout styles
- `web/app/client/design/app/elements.css` — Base element styles

*Client-side router* (adapted from agent-lab):
- `web/app/client/router/router.ts` — History API router with `<base href>` awareness, `:param` patterns
- `web/app/client/router/routes.ts` — Route definitions: `''` → `hd-documents-view`, `'prompts'` → `hd-prompts-view`, `'documents/:documentId/review'` → `hd-review-view`, `'*'` → `hd-not-found-view`
- `web/app/client/router/types.ts` — `RouteConfig`, `RouteMatch`
- `web/app/client/router/index.ts` — Public re-exports

*Core API layer*:
- `web/app/client/core/api.ts` — `Result<T>` discriminated union, `request()`, `stream()` (SSE wrapper), pagination types

*Placeholder views*:
- `web/app/client/views/not-found-view.ts` — `hd-not-found-view` component
- `web/app/client/views/documents-view.ts` — `hd-documents-view` placeholder stub
- `web/app/client/views/prompts-view.ts` — `hd-prompts-view` placeholder stub
- `web/app/client/views/review-view.ts` — `hd-review-view` placeholder stub

*Entry point*:
- `web/app/client/app.ts` — Imports `design/index.css` (side-effect), imports views, instantiates Router, calls `start()`

**Component pattern** (with CSS module plugin):
```typescript
import { LitElement, html } from 'lit';
import { customElement } from 'lit/decorators.js';
import styles from './my-view.css' with { type: 'css' };

@customElement('hd-my-view')
export class MyView extends LitElement {
  static styles = styles; // CSSStyleSheet — no unsafeCSS()
}
```

**Dependencies**: None

---

### Sub-issue 3: Go Web App Module, Server Integration, and Dev Experience

**Labels**: `feature`, `infrastructure`
**Milestone**: `v0.3.0 - Web Client`

Wire the Go side: embed built assets, serve via template set and module system, configure dev workflow tooling.

**Scope**:
- `web/app/app.go` — `//go:embed dist/*`, `//go:embed server/layouts/*`, `NewModule(basePath)` using `pkg/web/` TemplateSet and static serving
- `web/app/server/layouts/app.html` — HTML shell template with `<base href>`, CSS link, JS module script, app header with nav, `#app-content` container
- `cmd/server/modules.go` — Add `App *module.Module` to `Modules` struct, mount alongside API module
- `.air.toml` — Air configuration watching `cmd/`, `internal/`, `pkg/`, `web/app/dist/` for Go rebuild/restart
- `.mise.toml` — Add `web:build` and `web:watch` tasks

**Dependencies**: Sub-issue 1 (`pkg/web/`), Sub-issue 2 (produces `dist/` output to embed)

---

### Sub-issue 4: Web Development Skill

**Labels**: `documentation`
**Milestone**: `v0.3.0 - Web Client`

Document Herald-specific web development conventions as a Claude skill.

**Scope**:
- `.claude/skills/web-development/SKILL.md` — Conventions for `hd-` prefix, CSS module plugin (`{ type: 'css' }` → `CSSStyleSheet`), design tokens, three-tier component hierarchy (View → Stateful Component → Pure Element), `@lit/context` service pattern, signal-based reactivity

**Dependencies**: Sub-issues 1–3 (documents what was built)

---

## Dependency Graph

```
Sub-issue 1 (pkg/web/)     Sub-issue 2 (Client App)
         \                      /
          v                    v
       Sub-issue 3 (Go Integration)
                |
                v
       Sub-issue 4 (Skill)
```

Sub-issues 1 and 2 can proceed in parallel. Sub-issue 3 requires both. Sub-issue 4 comes last.

## Project Documentation Updates

After sub-issue creation:
- **`_project/objective.md`** — Create with scope, sub-issues table, architecture decisions
- **`_project/phase.md`** — Update CSS import constraint from `{ type: 'text' }` to `{ type: 'css' }`
- **`_project/README.md`** — Update Web Client section to reflect `{ type: 'css' }` + `CSSStyleSheet` approach (replace `with { type: 'text' }` references)
- **Issue #57 body** — Update CSS references from `{ type: 'text' }` to `{ type: 'css' }` and remove `unsafeCSS()` mentions

## Execution Plan

1. Create sub-issues 1–4 on `JaimeStill/herald` with bodies containing Context/Scope/Approach/Acceptance Criteria
2. Link each as sub-issue of #57 via GraphQL API
3. Add each to project board, assign Phase 3
4. Create `_project/objective.md`
5. Update `_project/phase.md`, `_project/README.md`, and issue #57 body with CSS approach correction
