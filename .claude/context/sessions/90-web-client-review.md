# 90 — Web Client Review & Optimization

## Summary

Holistic review of the web client architecture in its final Objective 5 state. Identified and extracted duplicated CSS patterns into shared styles, fixed component architecture friction points, and updated the web-development skill documentation to reflect the current codebase. No new features — purely optimization and documentation alignment.

## Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Shared style granularity | 5 modules (badge, buttons, cards, inputs, labels) | Each targets a distinct, frequently duplicated pattern. Avoids a single monolithic shared file. |
| Button color variants | `.btn-blue`, `.btn-green`, `.btn-red`, `.btn-yellow`, `.btn-muted` in `buttons.module.css` | Replaces 14+ component-specific button color classes with 5 composable variants |
| Upload `.field-input` not extracted | Kept component-specific | Different sizing/background from the shared `.input` pattern — forced extraction would require overrides |
| `querySelector` typing | Remove `<any>`, rely on `HTMLElementTagNameMap` | Tag name string literals already provide correct return types via the global interface declarations |
| Barrel exports | Named exports only, no `export *` | Explicit exports make the public API clear and prevent accidental re-exports |
| `defaults-content h4` not shared | Kept in prompt-form CSS | Content headings are semantically different from form labels despite identical typography |
| Disable validated classification actions | `?disabled` bound to `!!classification.validated_by` | Prevents re-validation or update of already-validated classifications |

## Files Modified

### New Files
- `app/client/design/styles/inputs.module.css` — shared form input styles
- `app/client/design/styles/labels.module.css` — shared section label typography
- `app/client/design/styles/cards.module.css` — shared card container pattern
- `.claude/skills/web-development/references/lifecycles.md` — Lit lifecycle hooks reference

### Modified Files — Source
- `app/client/design/styles/buttons.module.css` — added `.btn-blue`, `.btn-green`, `.btn-red`, `.btn-yellow`, `.btn-muted`
- `app/client/ui/elements/document-card.ts` + `.module.css` — shared styles, external_id/platform in meta
- `app/client/ui/elements/prompt-card.ts` + `.module.css` — shared styles
- `app/client/ui/elements/confirm-dialog.ts` + `.module.css` — shared button variant
- `app/client/ui/modules/classification-panel.ts` + `.module.css` — shared styles, disable when validated
- `app/client/ui/modules/document-grid.ts` + `.module.css` — shared styles
- `app/client/ui/modules/prompt-list.ts` + `.module.css` — shared styles
- `app/client/ui/modules/prompt-form.ts` + `.module.css` — shared styles
- `app/client/ui/modules/document-upload.ts` + `.module.css` — shared button variants
- `app/client/ui/views/documents-view.ts` — querySelector typing fix
- `app/client/ui/views/prompts-view.ts` — querySelector typing fix
- `app/client/domains/prompts/index.ts` — named exports
- `app/client/domains/storage/index.ts` — named exports

### Modified Files — Documentation
- `.claude/skills/web-development/SKILL.md` — directory tree, lifecycles reference link
- `.claude/skills/web-development/references/build.md` — path aliases
- `.claude/skills/web-development/references/css.md` — layer count, shared styles section
- `.claude/skills/web-development/references/services.md` — import paths, barrel convention
- `.claude/skills/web-development/references/state.md` — event name, querySelector
- `.claude/skills/web-development/references/components.md` — querySelector

## Patterns Established

- Shared styles compose with component CSS via `static styles` arrays — shared provides base, component retains layout overrides
- Button color variants use semantic names (`.btn-blue`) not domain names (`.classify-btn`) for reusability
- `querySelector` on tag name strings returns typed results via `HTMLElementTagNameMap` — no `<any>` needed
- Domain barrel exports use explicit named exports, never `export *`

## Validation Results

- `bun run build` succeeds
- `go vet ./...` passes
- Playwright browser verification: all three views render correctly with shared styles, disabled validation buttons confirmed, external_id/platform metadata visible on document cards
