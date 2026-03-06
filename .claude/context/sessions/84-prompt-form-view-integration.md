# 84 - Prompt Form Module and View Integration

## Summary

Created the `hd-prompt-form` module and replaced the stub `hd-prompts-view` with a full split-panel layout composing the prompt list and form. Added a default prompt reference panel (specification + default instructions) with a Go-side `?default=true` query param on the instructions endpoint. Established custom event naming convention. Removed unused `@lit-labs/signals` and `@lit/context` dependencies.

## Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Event naming convention | Simple action verbs (`select`, `delete`, `save`, `create`, `cancel`) | Component tag provides domain context; avoids redundant prefixing. Avoid overwriting semantically associated native events. |
| Card select event payload | Full `Prompt` object instead of ID | Eliminates unnecessary `PromptService.find()` call in the view |
| List `selectedId` → `selected` | `Prompt \| null` property | Richer type, view passes prompt directly |
| Disabled select + FormData | Fallback to `this.prompt.stage` | Disabled form elements excluded from FormData per HTML spec |
| Default prompt panel | Single `<details>` with spec + default instructions | Gives users full context of what they're overriding |
| Instructions endpoint `?default=true` | Bypass DB in handler, call hardcoded `Instructions()` directly | Backwards-compatible, avoids new system interface method |
| Remove signals/context deps | Deleted `@lit-labs/signals` and `@lit/context` | Unused — architecture uses `@state()` and direct service calls |
| Focus ring fix | `padding-inline` / `padding` on scroll containers | Gives outline-offset room to render without removing overflow containment |
| Toolbar layout | 4-column CSS grid | Search spans 3, new button 1; stage filter and sort each span 2 |

## Files Modified

- `app/client/ui/modules/prompt-form.ts` — new: create/edit form module with default prompt reference
- `app/client/ui/modules/prompt-form.module.css` — new: form styling
- `app/client/ui/modules/prompt-list.ts` — event renames, `selected` property change
- `app/client/ui/modules/prompt-list.module.css` — grid toolbar layout, padding fix
- `app/client/ui/modules/index.ts` — added `PromptForm` export
- `app/client/ui/elements/prompt-card.ts` — select event emits full prompt
- `app/client/ui/views/prompts-view.ts` — full split-panel view composition
- `app/client/ui/views/prompts-view.module.css` — split-panel layout
- `app/client/domains/prompts/service.ts` — `defaultOnly` param on instructions method
- `app/package.json` — removed unused dependencies
- `app/bun.lock` — updated lockfile
- `internal/prompts/handler.go` — `?default=true` query param support
- `tests/prompts/handler_test.go` — new test for default=true bypass
- `_project/api/prompts/README.md` — documented default query param
- `_project/api/prompts/prompts.http` — added default-only request
- `.claude/skills/web-development/SKILL.md` — custom event convention, directory listing

## Patterns Established

- **Custom event naming**: Simple action verbs, no domain prefix. Avoid overwriting native events semantically associated with the component. Documented in web-development skill.
- **Focus ring breathing room**: Use `padding-inline` or `padding` on scroll containers rather than removing `overflow: hidden` to give `outline-offset` room to render.
- **Default prompt reference panel**: Collapsible `<details>` showing immutable spec + hardcoded instructions for user context when editing prompts.

## Validation Results

- `bun run build` — clean
- `go vet ./...` — clean
- `go test ./tests/...` — all 20 packages pass (including new `default=true` handler test)
- Manual browser testing confirmed all acceptance criteria
