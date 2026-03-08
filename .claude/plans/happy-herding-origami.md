# Task #90 — Web Client Review & Optimization

## Context

This is the final task in Objective 5 (Document Review View). The review view composition was completed as a remediation in #89. This session focuses on a holistic review of the web client, identifying optimization opportunities, and producing an implementation guide for source code changes. Documentation/skill updates are applied directly (not in the guide).

## Execution Strategy

Two phases: (1) implementation guide for source code changes, (2) documentation/skill updates reflecting the final implementation.

---

## Phase 1: Implementation Guide (Source Code Changes)

### 1A. Shared Styles — New CSS Modules

Extract duplicated CSS patterns into `app/client/design/styles/`:

**`inputs.module.css`** — Form input base styles
- Classes: `.input` (covers `input`, `select`, `textarea`)
- Extracted from: `classification-panel.module.css:140-159`, `prompt-form.module.css:51-71`, `document-grid.module.css:17-32`, `prompt-list.module.css:17-32`, `document-upload.module.css:129-147`
- Includes: padding, border, border-radius, background, color, font, focus-visible, disabled states

**`labels.module.css`** — Section label typography
- Classes: `.label`
- Extracted from: `classification-panel.module.css:37-43,132-138`, `prompt-form.module.css:43-49,108-113`
- Includes: text-xs, font-mono, color-1, uppercase, letter-spacing

**`cards.module.css`** — Card container pattern
- Classes: `.card`
- Extracted from: `document-card.module.css:5-14`, `prompt-card.module.css:5-14` (identical)
- Includes: flex column, gap-3, padding-4, bg-1, border, radius-md, transition

**Button color variants in `buttons.module.css`**
- Add: `.btn-blue`, `.btn-green`, `.btn-red`, `.btn-yellow`, `.btn-muted`
- Pattern: `border-color`, `color`, `:hover { background }` with `:not(:disabled)` guard
- Replaces: `.classify-btn`, `.review-btn`, `.delete-btn`, `.validate-btn`, `.update-btn`, `.cancel-btn`, `.save-btn`, `.toggle-btn`, `.upload-btn`, `.clear-btn`, `.bulk-btn`, `.new-btn`, `.confirm-btn`, `.remove-btn`

### 1B. Component CSS — Consume Shared Styles

Update each component to import and use the new shared styles, removing duplicated CSS:

| Component | Import | Replace |
|-----------|--------|---------|
| `document-card.ts` + `.module.css` | `cardStyles` | Remove `.card` block, use `.btn-blue`, `.btn-green`, `.btn-red` |
| `prompt-card.ts` + `.module.css` | `cardStyles` | Remove `.card` block, use `.btn-green`, `.btn-yellow`, `.btn-red` |
| `confirm-dialog.ts` + `.module.css` | — | Use `.btn-red` |
| `document-grid.ts` + `.module.css` | `inputStyles` | Remove `.search-input/.filter-select/.sort-select` base block, use `.btn-blue` |
| `prompt-list.ts` + `.module.css` | `inputStyles` | Remove `.search-input/.filter-select/.sort-select` base block, use `.btn-blue` |
| `classification-panel.ts` + `.module.css` | `inputStyles`, `labelStyles` | Remove `.field input/textarea`, `.field label`, `.section-label` base blocks, use `.btn-green`, `.btn-blue`, `.btn-muted` |
| `prompt-form.ts` + `.module.css` | `inputStyles`, `labelStyles` | Remove `.field input/select/textarea`, `.field label`, `.defaults-content h4` base blocks, use `.btn-green`, `.btn-muted` |
| `document-upload.ts` + `.module.css` | — | Use `.btn-blue`, `.btn-yellow`, `.btn-red` |

### 1C. Classification Panel — Disable Actions When Validated

In `classification-panel.ts`, disable the Validate and Update buttons when `this.classification?.validated_by` is truthy. The `renderViewMode()` method (line 188-198) should add `?disabled` bindings.

### 1D. Document Card — Show External ID and Platform

In `document-card.ts`, add `external_id` and `external_platform` to the card's meta section. These fields exist on the `Document` type already (`document.ts:11-12`).

### 1E. Fix `querySelector<any>` Type Erasure

In `documents-view.ts:16` and `prompts-view.ts:33`, replace `querySelector<any>` with proper type assertions using `HTMLElementTagNameMap`:

```typescript
// documents-view.ts
this.renderRoot.querySelector("hd-document-grid")?.refresh();
// prompts-view.ts
this.renderRoot.querySelector("hd-prompt-list")?.refresh();
```

`querySelector` with the tag name string already returns the correct type via `HTMLElementTagNameMap` declaration — no `<any>` or cast needed.

### 1F. Domain Barrel Exports — Named Exports Only

Fix `prompts/index.ts` and `storage/index.ts` to use named exports instead of `export *`:

```typescript
// prompts/index.ts
export type { Prompt, PromptStage, StageContent, CreatePromptCommand, UpdatePromptCommand, SearchRequest } from "./prompt";
export { PromptService } from "./service";

// storage/index.ts
export type { BlobMeta, BlobList } from "./blob";
export { StorageService } from "./service";
export type { StorageListParams } from "./service";
```

Then verify no consumers rely on barrel imports that would break.

---

## Verification

1. `bun run build` succeeds
2. `go vet ./...` passes
3. Visual verification:
   - Documents view: cards render with external_id/platform, button colors match
   - Prompts view: form inputs, labels, button colors match
   - Review view: classification panel validate/update disabled when validated
   - All views: no visual regressions from CSS extraction

## Files Modified

**New files:**
- `app/client/design/styles/inputs.module.css`
- `app/client/design/styles/labels.module.css`
- `app/client/design/styles/cards.module.css`

**Modified files:**
- `app/client/design/styles/buttons.module.css` — add color variants
- `app/client/ui/elements/document-card.ts` + `.module.css` — external_id/platform, shared styles
- `app/client/ui/elements/prompt-card.ts` + `.module.css` — shared styles
- `app/client/ui/elements/confirm-dialog.module.css` — shared button variant
- `app/client/ui/modules/classification-panel.ts` + `.module.css` — disable when validated, shared styles
- `app/client/ui/modules/document-grid.ts` + `.module.css` — shared styles
- `app/client/ui/modules/prompt-list.ts` + `.module.css` — shared styles
- `app/client/ui/modules/prompt-form.ts` + `.module.css` — shared styles
- `app/client/ui/modules/document-upload.module.css` — shared button variants
- `app/client/ui/views/documents-view.ts` — fix querySelector typing
- `app/client/ui/views/prompts-view.ts` — fix querySelector typing
- `app/client/domains/prompts/index.ts` — named exports
- `app/client/domains/storage/index.ts` — named exports

---

## Phase 2: Documentation & Skill Updates (Post-Implementation)

Applied after implementation to reflect the actual final state of the codebase:

- `.claude/skills/web-development/SKILL.md` — update directory tree with all current components
- `.claude/skills/web-development/references/build.md` — fix stale `@app/*` alias → current path aliases
- `.claude/skills/web-development/references/css.md` — fix "three layers" → "four layers", update code example, document new shared style modules
- `.claude/skills/web-development/references/services.md` — fix `@app/` import paths → `@core`/`@domains/*`, update barrel export convention (named exports only)
- `.claude/skills/web-development/references/state.md` — fix `prompt-select` → `select` event name
- `.claude/skills/web-development/references/components.md` — update querySelector examples to reflect proper typing, update code examples for shared styles
- `.claude/skills/web-development/references/lifecycles.md` (new) — Lit lifecycle hook patterns: `connectedCallback`, `disconnectedCallback`, `updated`, `willUpdate`
