# 83 - Prompt List Module with Stage Filtering and Activation

## Summary

Created the `hd-prompt-list` stateful module for browsing, filtering, pagination, and prompt lifecycle management (activate/deactivate/delete). Also cleaned up `hd-document-grid` to delegate filter/sort/search handlers to `refresh()` instead of duplicating page reset + fetch logic.

## Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Search method | `PromptService.search()` (POST) | Server-side filtering with `SearchRequest` body, consistent with document-grid pattern |
| Layout | Vertical flex list | Prompts are text-heavy; vertical list is more scannable than grid (per objective decision) |
| Debounced search delegation | `refresh()` | Eliminates duplicated `page = 1` + fetch logic; applied to both prompt-list and document-grid |
| Confirm dialog message | Extracted to local variable | Linter reformatted inline template expression; cleaner as variable |

## Files Modified

- `app/client/ui/modules/prompt-list.ts` (new)
- `app/client/ui/modules/prompt-list.module.css` (new)
- `app/client/ui/modules/index.ts` (added PromptList export)
- `app/client/ui/modules/document-grid.ts` (cleanup: delegate to refresh())

## Patterns Established

- Module filter/sort/search handlers should delegate to `refresh()` rather than duplicating `page = 1` + fetch

## Validation Results

- Bun build: pass
- go vet: pass
- Fixed typo: `?selectetd` → `?selected` in sort select
