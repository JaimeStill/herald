# 82 - Prompt Card Pure Element and Search Request Type

## Summary

Created the `hd-prompt-card` pure element and `SearchRequest` type for the prompts domain. Also flattened the `ui/` directory structure (removing domain subdirectories), inverted the router's route dependency, and fixed a pre-existing app test failure caused by HTML self-closing tag formatting.

## Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Flatten UI tiers | Components directly in `ui/elements/`, `ui/modules/`, `ui/views/` | `hd-` prefix already namespaces by domain; subdirectories added unnecessary nesting and barrel files |
| Router dependency inversion | Routes injected via constructor, defined in `app/client/routes.ts` | Router shouldn't know about specific routes; co-locating routes with `app.ts` improves discoverability |
| Stage badge colors | classify=blue, enhance=orange, finalize=green | Reuses existing color tokens; enhance (orange) is visually distinct from the blue/green pair |
| Badge consolidation | Group by color, not by variant name | Reduces CSS rules; classify/finalize/review/uploading all share blue |
| SearchRequest self-contained | All fields defined directly, no `PageRequest` extension | Follows documents domain convention; enables eventual `PageRequest` removal from `@core` |
| App test fix | Match `<base href="/app/"` without closing tag | Prettier reformats self-closing tags; test should be format-agnostic |

## Files Modified

- `app/client/domains/prompts/prompt.ts` — added `SearchRequest` interface
- `app/client/domains/prompts/service.ts` — migrated `list()` and `search()` to `SearchRequest`
- `app/client/design/styles/badge.module.css` — consolidated and added stage variants
- `app/client/ui/elements/prompt-card.ts` — **new** pure element
- `app/client/ui/elements/prompt-card.module.css` — **new** component styles
- `app/client/ui/elements/index.ts` — rewritten (flat barrel)
- `app/client/ui/modules/index.ts` — rewritten (flat barrel)
- `app/client/ui/views/index.ts` — rewritten (flat barrel)
- `app/client/core/router/router.ts` — accepts routes via constructor
- `app/client/routes.ts` — **new** (moved from `core/router/routes.ts`)
- `app/client/app.ts` — imports routes, passes to Router constructor
- `tests/app/app_test.go` — fixed base href assertion to be format-agnostic
- `.claude/skills/web-development/SKILL.md` — updated directory structure and router description
- `.claude/skills/web-development/references/router.md` — updated for dependency inversion

## Patterns Established

- **Flat UI tiers**: No domain subdirectories in `ui/elements/`, `ui/modules/`, `ui/views/`. Components and their CSS modules live directly in the tier directory.
- **Router route injection**: Routes defined at `app/client/routes.ts` and passed to `Router` constructor. Router package has no knowledge of specific routes.
- **Domain-owned SearchRequest**: Each domain defines its own `SearchRequest` with all fields (no `PageRequest` extension), enabling eventual removal of shared `PageRequest`.

## Validation Results

- `bun run build` — passes
- `go vet ./...` — clean
- `go test ./tests/...` — all 20 packages pass (including fixed app tests)
