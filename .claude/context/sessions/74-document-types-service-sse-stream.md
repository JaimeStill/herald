# 74 - Document types, service, and SSE stream enhancement

## Summary

Established the web client's foundational data layer: enhanced SSE streaming with event type parsing and POST support, TypeScript domain types mirroring all Go API response shapes, and stateless service objects mapping every Go handler endpoint across four domains (documents, classifications, prompts, storage). Also established the domain directory convention and simplified the component/state architecture — components call services directly with no orchestration middleman.

## Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Domain directory convention | Types and services in `app/client/<domain>/`, separate from `views/` | Domain infrastructure is shared across views; co-locating with views creates coupling |
| Service/state separation | Stateless services + components call services directly | Eliminates unnecessary orchestration layer; components already know when to call services |
| No state files | Shared data via `Signal.State` + `@provide`/`@consume` inline; local state via `@state()` | Factory functions with methods were pass-through wrappers that added complexity without value |
| Encapsulated streaming | Document component owns classify SSE lifecycle, emits completion event | Per-document concern; parent view only cares about outcome, not intermediate progress |
| Full endpoint coverage | All handler methods mapped, not just documents view needs | Services are shared infrastructure; partial coverage would require revisiting later |
| PascalCase service objects | `DocumentService`, `ClassificationService`, etc. | Matches Go naming convention; clearly distinguishes services from local variables |
| JSDoc on public TypeScript API | Same responsibility as Godoc on Go exports | Provides IDE intellisense for foundational infrastructure used across all views |

## Files Modified

- `app/client/core/api.ts` — Enhanced `stream()` with `onEvent(type, data)`, `init?: RequestInit`; added `ExecutionEvent`; JSDoc
- `app/client/core/index.ts` — Re-export `ExecutionEvent`, `StreamOptions`
- `app/client/documents/document.ts` — `Document`, `DocumentStatus` types
- `app/client/documents/service.ts` — `DocumentService` (5 methods)
- `app/client/documents/index.ts` — Barrel
- `app/client/classifications/classification.ts` — `Classification` type
- `app/client/classifications/service.ts` — `ClassificationService` (8 methods), `ValidateCommand`, `UpdateCommand`
- `app/client/classifications/index.ts` — Barrel
- `app/client/prompts/prompt.ts` — `Prompt`, `PromptStage`, `StageContent`, command types
- `app/client/prompts/service.ts` — `PromptService` (11 methods)
- `app/client/prompts/index.ts` — Barrel
- `app/client/storage/blob.ts` — `BlobMeta`, `BlobList` types
- `app/client/storage/service.ts` — `StorageService` (3 methods), `StorageListParams`
- `app/client/storage/index.ts` — Barrel
- `app/client/router/types.ts` — JSDoc added
- `app/client/router/router.ts` — JSDoc added
- `app/client/router/routes.ts` — JSDoc added
- `app/client/css.d.ts` — JSDoc added
- `.claude/CLAUDE.md` — Added JSDoc convention alongside Godoc
- `.claude/skills/web-development/SKILL.md` — Updated architecture (services/state, component hierarchy, anti-patterns)
- `.claude/skills/web-development/references/services.md` — Rewritten for stateless service pattern
- `.claude/skills/web-development/references/state.md` — New: shared reactive state, encapsulated streaming
- `.claude/skills/web-development/references/components.md` — Rewritten: components call services directly
- `.claude/skills/web-development/references/api.md` — Rewritten: new stream() API, event parsing

## Patterns Established

- **Domain directory convention**: `app/client/<domain>/` holds types + stateless service, separate from views
- **Stateless service pattern**: PascalCase objects with `base` path, methods return `Result<T>` or `AbortController`
- **Components call services directly**: No orchestration layer; views update signals, components use `@state()` for local concerns
- **Encapsulated streaming**: Component owning the SSE concern manages its own lifecycle and emits completion events
- **JSDoc as AI responsibility**: Added to `.claude/CLAUDE.md` alongside Godoc convention

## Validation Results

- `bun run build` compiles without errors
- All 14 new files verified with correct exports
- All 4 services map every Go handler endpoint (27 methods total)
- `stream()` correctly parses `event:` + `data:` line pairs
- Web-development skill fully updated with simplified architecture
