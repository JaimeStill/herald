# Session: 77 - Document Grid, View Integration, and Bulk Classify

## Summary

Final sub-issue of Objective #59 (Document Management View). Assembled the full document management interface by composing the service layer (#74), card + progress elements (#75), and upload component (#76). Extensive Phase 5 testing with the developer drove significant design refinements and culminated in a major client directory restructure.

## What Was Done

### Core Implementation
- **SearchRequest convention**: Each domain defines its own `SearchRequest` interface (pagination + domain-specific filters). `toQueryString` generalized to `<T extends object>`.
- **Document grid module** (`hd-document-grid`): Toolbar with debounced search, status filter, sort. Responsive CSS grid (`auto-fill`). SSE classify orchestration with per-card progress. Bulk select + classify via `Promise.allSettled`. Delete with confirmation dialog. Pagination via `hd-pagination`.
- **Pagination element** (`hd-pagination`): Reusable pure element, always renders when documents exist.
- **Confirm dialog element** (`hd-confirm-dialog`): Reusable overlay with message, confirm/cancel actions. Used for delete confirmation with document filename.
- **Documents view** (`hd-documents-view`): Composition root with upload toggle, grid, and refresh wiring.

### Design System Improvements
- **`light-dark()` consolidation**: Replaced duplicated `@media (prefers-color-scheme)` blocks in tokens.css.
- **Shared styles**: `buttons.module.css` and `badge.module.css` in `design/styles/`, imported via `@styles/*` alias.
- **Monospace on interactive elements**: Buttons, inputs, selects, drop zone text use `--font-mono`. Reset layer sets it globally for light DOM.
- **Button refinements**: `--bg` background (not `--bg-2`), no border color change on hover.
- **Card selection**: Border color change (`--blue`) on click instead of checkbox. Header is click target.
- **`prefers-reduced-motion`** guard on classify progress pulse animation.
- **View Transitions API** progressive enhancement in router.

### CSS Modules Plugin Fix
- Backslash escaping added to `css-modules.ts` — CSS escape sequences like `\00b7` (middle dot) were being mangled in template literal embedding.

### Bug Fixes
- **Document Create scan mismatch**: `RETURNING` clause padded with `NULL, NULL, NULL` for classification join fields.
- **CSS Grid `auto-fill`**: Fixed typo `autofill` → `auto-fill` that caused single-column layout.
- **Pagination `static styles` typo**: `stlyes` → `styles` prevented shared button styles from applying.

### Client Directory Restructure (#77)
Major reorganization for scalability:
- `core/` — framework utilities: api, router, formatting
- `design/` — global design system: core/, styles/, app/
- `domains/` — data types + services: classifications, documents, prompts, storage
- `ui/` — all rendering: elements/ (pure), modules/ (stateful), views/ (route-level)

**Path aliases**: `@core`, `@core/*`, `@design/*`, `@domains/*`, `@styles/*`, `@ui/*`

**Component vocabulary**: View → Module → Element. "Module" replaces "component" for the stateful tier (self-contained capability unit that owns state and service interfaces).

**Import convention** established: (1) third-party, (2) cross-package aliased, (3) relative, (4) styles. Infra before types, sorted by path depth then alphabetically.

### Documentation
- Web development skill (`SKILL.md`) rewritten to reflect restructure, new vocabulary, import convention, and design patterns.
- Memory updated with consolidated web client section.
- JSDoc added on all exported TypeScript surfaces.

## Files Changed

### New Files
- `app/client/ui/modules/documents/document-grid.ts` + `.module.css`
- `app/client/ui/elements/pagination/pagination-controls.ts` + `.module.css` + `index.ts`
- `app/client/ui/elements/dialog/confirm-dialog.ts` + `.module.css` + `index.ts`
- `app/client/design/styles/buttons.module.css`
- `app/client/design/styles/badge.module.css`
- `app/client/view-transitions.d.ts`

### Modified Files
- `app/tsconfig.json` — new path aliases
- `app/plugins/css-modules.ts` — backslash escaping fix
- `app/client/app.ts` — restructured imports
- `app/client/core/api.ts` — generic `toQueryString`
- `app/client/core/index.ts` — barrel updates
- `app/client/design/core/tokens.css` — `light-dark()` consolidation
- `app/client/design/core/reset.css` — monospace on interactive elements
- `app/client/design/index.css` — layer declaration update
- `app/client/design/app/app.css` — `@layer app` wrapper
- `app/client/domains/documents/document.ts` — `SearchRequest` type
- `app/client/domains/documents/service.ts` — accepts `SearchRequest`
- `app/client/domains/*/service.ts` — import path updates
- `app/client/ui/elements/documents/document-card.ts` + `.module.css` — selection, delete, classification display, shared styles
- `app/client/ui/elements/documents/classify-progress.module.css` — reduced-motion guard
- `app/client/ui/modules/documents/document-upload.ts` + `.module.css` — queue header actions, shared styles, monospace
- `app/client/ui/views/documents/documents-view.ts` + `.module.css` — composition root
- `app/client/ui/modules/documents/document-grid.module.css` — toolbar, grid layout
- `internal/documents/repository.go` — RETURNING clause fix
- `.mise.toml` — `web:fmt` task
- `_project/README.md` — PageRequest cleanup note

### Deleted Files
- `app/client/design/app/elements.css` (empty)
- `app/client/components/` (moved to `ui/modules/`)
- `app/client/elements/` (moved to `ui/elements/`)
- `app/client/views/` (moved to `ui/views/`)
- `app/client/classifications/` (moved to `domains/`)
- `app/client/documents/` (moved to `domains/`)
- `app/client/prompts/` (moved to `domains/`)
- `app/client/storage/` (moved to `domains/`)
- `app/client/router/` (moved to `core/router/`)
- `app/client/formatting/` (moved to `core/formatting/`)
