# Phase Planning: Phase 3 — Web Client (v0.3.0)

## Context

Phases 1 (Service Foundation) and 2 (Classification Engine) are complete. Herald has a fully operational Go API with documents, classifications, prompts, and storage endpoints, a 4-node classification workflow with observer injection, and Docker Compose local infrastructure. Phase 3 adds the embedded Lit 3.x web client with three views, replaces Vite with native Bun builds, adds Air for hot reload, and implements SSE streaming for classification progress.

## Session Metadata

- **Phase name**: Phase 3 - Web Client
- **Version target**: v0.3.0
- **Primary repository**: JaimeStill/herald
- **Milestone**: `v0.3.0 - Web Client` (exists, open, #3)
- **Project board phase option**: `Phase 3 - Web Client` (ID: `55e651bc`)

## Pre-Planning Cleanup

- Update Objective #27 (Classifications Domain) board status from "In Progress" to "Done" (issue is closed, board is stale)
- Update `_project/README.md` Phase 3 description: replace "Bun + Vite" with "Bun" and add Air mention

## Key Design Decisions

### Bun-Native Builds (No Vite)

The CSS import challenge is resolved via ES import attributes:

- **Global CSS**: Side-effect import in app.ts (`import './design/index.css'`). Bun extracts it into `dist/app.css` alongside `dist/app.js` — same behavior as Vite.
- **Component CSS**: Text import (`import styles from './component.css' with { type: 'text' }`). Bun returns CSS as a string for `unsafeCSS()` — replaces Vite's `?inline` query.
- **Build scripts**: TypeScript scripts using `Bun.build()` API replace `vite.config.ts`.
- **CSS cascade layers**: `@layer` declarations and `@import` work natively — no Vite processing needed.

### Development Experience (Air + Bun Watch)

Two-terminal workflow:
1. **Terminal 1**: `bun run watch` — Bun watches `client/**/*.{ts,css}`, rebuilds `dist/` on change
2. **Terminal 2**: `air` — Air watches Go files + `web/app/dist/`, rebuilds and restarts the server

Flow: edit TS/CSS → Bun rebuilds dist → Air detects dist change → server restarts → browser refresh. Edit Go → Air rebuilds server → server restarts.

### SSE Endpoint Design

Separate SSE endpoint (`GET /api/classifications/{documentId}/stream`) that initiates classification and streams progress. The existing `POST /api/classifications/{documentId}` remains for non-streaming callers. EventSource only supports GET, and the unidirectional server-push pattern is established in `_project/README.md`.

### Web Client Structure

```
web/app/
├── app.go                    # Go: //go:embed, NewModule()
├── package.json              # Bun project (no Vite)
├── tsconfig.json
├── scripts/
│   ├── build.ts              # Bun.build() production
│   └── watch.ts              # Bun.build() + file watcher
├── client/
│   ├── app.ts                # Entry: imports design CSS, starts router
│   ├── css.d.ts              # TS declarations for CSS text imports
│   ├── router/               # Client-side History API router
│   ├── core/                 # API client (Result<T>), pagination, streaming
│   ├── design/               # CSS cascade layers, tokens, reset, theme
│   ├── views/                # Stub views (not-found)
│   ├── documents/            # Document domain (service, types, components)
│   ├── prompts/              # Prompts domain (service, types, components)
│   └── review/               # Review domain (service, types, components)
├── dist/                     # Build output: app.js, app.css
└── server/
    └── layouts/
        └── app.html          # Go template shell
```

## Objectives

### Objective 1: Web Client Foundation and Build System

**What**: Everything needed to serve a working Lit shell from the Go binary with Bun-native builds, Air hot reload, and the web-development skill.

**Scope**:
- `pkg/web/` — TemplateSet, Router, static file serving (adapt from `~/code/agent-lab/pkg/web/`)
- `web/app/app.go` — Go module with `//go:embed`, `NewModule(basePath)`
- `web/app/server/layouts/app.html` — Shell template with `<base href>`, CSS link, JS module script
- `web/app/client/` — Entry point, CSS declarations, router (routes: `''`, `'prompts'`, `'documents/:documentId/review'`, `'*'`), core API layer (`Result<T>`, streaming utilities), design system (cascade layers, tokens, reset, theme), placeholder view stubs
- `web/app/package.json` + `tsconfig.json` — Bun project with Lit 3.x, @lit/context, @lit-labs/signals
- `web/app/scripts/build.ts` + `watch.ts` — Bun build scripts
- `cmd/server/modules.go` — Mount web app module alongside API module
- `.air.toml` — Air configuration
- `mise.toml` — Add `web:build`, `web:watch` tasks
- `.claude/skills/web-development/SKILL.md` — Herald-specific web dev skill (prefix `hd-`, Bun text imports, Herald tokens)

**Dependencies**: None (first objective)

**Critical files to modify**:
- `cmd/server/modules.go` — add App module
- `pkg/web/doc.go` → becomes full package

**Reference files**:
- `~/code/agent-lab/pkg/web/` (views.go, router.go, static.go)
- `~/code/agent-lab/web/app/app.go`
- `~/code/agent-lab/.claude/skills/web-development/SKILL.md`

---

### Objective 2: SSE Classification Streaming

**What**: Server-side streaming observer and SSE endpoint for classification workflow progress. Pure Go backend — no UI consumption yet.

**Scope**:
- `internal/workflow/observer.go` — `StreamingObserver` implementing observer interface, buffered event channel
- `internal/workflow/events.go` — `ExecutionEvent` type with stage_start, stage_complete, decision, complete, error event types
- `internal/workflow/workflow.go` — `Execute` accepts optional observer, passes to graph config
- `internal/classifications/` — `ClassifyStream` method and SSE handler at `GET /classifications/{documentId}/stream`
- API Cartographer update for SSE endpoint

**Dependencies**: None (parallel with Objective 1, pure backend)

**Critical files to modify**:
- `internal/workflow/workflow.go:46` — `cfg.Observer = "noop"` → configurable
- `internal/classifications/handler.go` — new SSE handler
- `internal/classifications/system.go` — new `ClassifyStream` method
- `internal/api/routes.go` — register new route

**Reference files**:
- `~/code/agent-lab/internal/workflows/streaming.go` (observer pattern)

---

### Objective 3: Document Management View

**What**: The primary view — document grid, upload (single + batch), classify trigger with SSE progress, search/filter, bulk operations.

**Scope**:
- `web/app/client/documents/` — types, service (signals for documents/loading/error/classifyingIds), DocumentService with list/upload/classify/classifyBulk/delete
- View: `hd-documents-view` (@provide DocumentService)
- Components: `hd-document-grid` (search, filter, pagination), `hd-document-card` (status badge, actions), `hd-document-upload` (multi-file via `<input multiple>`, Promise.allSettled), `hd-classify-progress` (SSE event rendering)
- Bulk classify: Promise.allSettled with per-document SSE connections

**Dependencies**: Objective 1 (foundation), Objective 2 (SSE endpoint)

---

### Objective 4: Prompt Management View

**What**: CRUD interface for named prompt overrides.

**Scope**:
- `web/app/client/prompts/` — types, service (signals for prompts/loading/error/selected), PromptService with list/find/create/update/delete/toggleActive
- View: `hd-prompts-view` (@provide PromptService, split layout)
- Components: `hd-prompt-list` (stage filter, active toggle), `hd-prompt-form` (create/edit, FormData extraction), `hd-prompt-card` (name, stage badge, active indicator)

**Dependencies**: Objective 1 (foundation)

---

### Objective 5: Document Review View

**What**: Side-by-side PDF viewer and classification record for validation or manual adjustment.

**Scope**:
- `web/app/client/review/` — types, service (signals for document/classification/loading/error), ReviewService with load/validate/update
- View: `hd-review-view` (@provide ReviewService, receives `documentId` from router, two-panel layout)
- Components: `hd-pdf-viewer` (renders PDF via storage API URL in `<iframe>`/`<object>`), `hd-classification-panel` (classification display, validate button, update form), `hd-markings-list` (styled tags for markings_found)
- Actions: Validate → `POST /api/classifications/{id}/validate`, Update → `PUT /api/classifications/{id}`

**Dependencies**: Objective 1 (foundation), Objective 3 (document types, navigation entry point)

## Dependency Graph

```
Objective 1 (Foundation)     Objective 2 (SSE)
        |          \              |
        v           v             v
  Objective 4    Objective 3 ◄────┘
  (Prompts)      (Documents)
                     |
                     v
               Objective 5
               (Review)
```

Objectives 1 and 2 can proceed in parallel. Objective 4 can start after Objective 1. Objective 3 requires both 1 and 2. Objective 5 requires 1 and 3.

## Summary

| # | Objective | Depends On |
|---|-----------|------------|
| 1 | Web Client Foundation and Build System | — |
| 2 | SSE Classification Streaming | — |
| 3 | Document Management View | 1, 2 |
| 4 | Prompt Management View | 1 |
| 5 | Document Review View | 1, 3 |

## Deliverables

After this session:
- 5 Objective issues created on JaimeStill/herald with `objective` label and milestone `v0.3.0 - Web Client`
- All objectives added to project board #7, assigned to Phase 3
- `_project/phase.md` updated with phase scope, objectives table, and constraints
- `_project/README.md` Phase 3 description updated (Bun-only, Air)
- Objective #27 board status corrected to Done

## Verification

- All 5 objectives visible on the project board under Phase 3
- `_project/phase.md` has the objectives table with status column
- Milestone `v0.3.0 - Web Client` has 5 open issues
- Each objective body has enough context for an objective planning session
