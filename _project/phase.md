# Phase 3 — Web Client

**Version Target**: v0.3.0
**Milestone**: v0.3.0 - Web Client

## Scope

Deliver a Lit 3.x web client embedded in the Go binary with three views (document management, prompt management, document review), native Bun builds replacing Vite, Air hot reload for development, and SSE streaming for classification workflow progress.

## Goals

- Establish Bun-native build infrastructure with ES import attributes for CSS text imports (no Vite dependency)
- Integrate Air for Go hot reload coordinated with Bun watch for TypeScript/CSS — two-terminal dev workflow
- Implement SSE streaming observer for classification workflow progress via `GET /api/classifications/{documentId}/stream`
- Build document management interface with upload (single + batch), classify (single + bulk via `Promise.allSettled`), search, filter, and real-time SSE progress
- Build prompt management interface with CRUD, stage filtering, and active toggle
- Build document review interface with side-by-side PDF viewer and classification validation/adjustment
- Create Herald-specific web-development skill (`.claude/skills/web-development/SKILL.md`)

## Objectives

| # | Objective | Issue | Status | Depends On |
|---|-----------|-------|--------|------------|
| 1 | Web Client Foundation and Build System | #57 | Open | — |
| 2 | SSE Classification Streaming | #58 | Open | — |
| 3 | Document Management View | #59 | Open | #57, #58 |
| 4 | Prompt Management View | #60 | Open | #57 |
| 5 | Document Review View | #61 | Open | #57, #59 |

## Dependency Graph

```
Objective 1 (Foundation)     Objective 2 (SSE)
        |          \              |
        v           v             v
  Objective 4    Objective 3 <----'
  (Prompts)      (Documents)
                     |
                     v
               Objective 5
               (Review)
```

Objectives 1 and 2 can proceed in parallel. Objective 4 can start after 1. Objective 3 requires both 1 and 2. Objective 5 requires 1 and 3.

## Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Build tool | Native Bun (no Vite) | `Bun.build()` API handles bundling natively. CSS text imports via `with { type: 'text' }` replace Vite's `?inline`. Eliminates a dependency and simplifies the toolchain. |
| Dev experience | Air + Bun watch (two terminals) | Air watches Go + dist/ for server rebuild. Bun watches client/ for asset rebuild. Clean separation of concerns. |
| SSE endpoint | Separate `GET` endpoint | EventSource requires GET. Existing `POST /classify` remains for non-streaming callers. Backward compatible. |
| Component prefix | `hd-` | Herald-specific, avoids collision with agent-lab's `lab-` prefix. |
| Client structure | Single `web/app/` package | Herald has one client (no multi-client abstraction needed). Build scripts, source, and Go embedding co-located. |

## Constraints

- Component CSS uses `import styles from './x.css' with { type: 'text' }` — not Vite's `?inline` query
- Global CSS uses side-effect import (`import './design/index.css'`) — Bun extracts to dist/app.css
- All views follow the three-tier hierarchy: View (@provide) → Stateful Component (@consume) → Pure Element (props/events)
- Services use Signal.State + @lit/context for reactive state management
- No shadow DOM bypass — use CSS custom properties for design token penetration
