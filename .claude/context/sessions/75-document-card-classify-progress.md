# 75 - Document Card and Classify Progress Elements

## Summary

Created two pure elements (`hd-document-card`, `hd-classify-progress`) for the documents view, along with supporting infrastructure (`WorkflowStage` domain type, `formatting/` module). Resolved a fundamental architectural contradiction in the web-development skill where the "Encapsulated Streaming" section showed a pure element calling `ClassificationService.classify()` directly, violating the three-tier hierarchy.

## Key Decisions

### Pure Element Import Boundary

Established a clear boundary: pure elements can import `lit`, their own CSS module, and **immutable domain infrastructure** (types, constants, formatters). Prohibited: services, signals, context (`@provide`/`@consume`), `SignalWatcher`, router utilities — anything that holds or mutates state.

### Streaming Orchestration Correction

Moved SSE lifecycle ownership from the pure element tier to the stateful component tier. The list/grid component owns `ClassificationService.classify()` calls, tracks per-item progress via `@state()` (Map), and passes progress data to pure cards as `@property()` values. Cards dispatch intent events only.

### WorkflowStage as Domain Infrastructure

Formalized workflow stages as a domain type (`WorkflowStage`) and constant (`WORKFLOW_STAGES`) in the classifications domain, rather than a local constant in the element. This follows the principle that domain knowledge (what stages exist) is immutable infrastructure importable by elements.

### Formatting Module

Created `app/client/formatting/` module parallel to Go `pkg/formatting/` with `formatBytes()` and `formatDate()` utilities. Shared across elements rather than implemented as private methods.

## Changes

### New Files
- `app/client/elements/documents/document-card.ts` — Pure element displaying document metadata with classify/review action buttons
- `app/client/elements/documents/document-card.module.css` — Card styles with status badge colors, action button accents
- `app/client/elements/documents/classify-progress.ts` — Pure element rendering horizontal 4-stage pipeline indicator
- `app/client/elements/documents/classify-progress.module.css` — Pipeline styles with pending/active/completed states
- `app/client/elements/documents/index.ts` — Domain barrel
- `app/client/elements/index.ts` — Top-level elements barrel
- `app/client/formatting/bytes.ts` — `formatBytes()` (base-1024 units)
- `app/client/formatting/date.ts` — `formatDate()` (locale-aware short date)
- `app/client/formatting/index.ts` — Formatting barrel

### Modified Files
- `app/client/classifications/classification.ts` — Added `WorkflowStage` type and `WORKFLOW_STAGES` constant
- `app/client/classifications/index.ts` — Export new type and constant
- `app/client/app.ts` — Added `import './elements'` for element registration
- `.claude/skills/web-development/SKILL.md` — Updated pure element tools, file structure, anti-patterns
- `.claude/skills/web-development/references/components.md` — Replaced examples with streaming orchestration and card patterns
- `.claude/skills/web-development/references/state.md` — Replaced "Encapsulated Streaming" with corrected "Streaming Orchestration" section

## Bugs Fixed
- Missing `.` prefix in CSS selector `stage.active .label` → `.stage.active .label`
- Unclosed `<hd-classify-progress>` tag in `renderProgress()` method
