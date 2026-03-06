# 84 - Prompt Form Module and View Integration

## Context

Third and final sub-issue of Objective #60 (Prompt Management View). The prompt card (#82) and prompt list (#83) are merged. This task creates the `hd-prompt-form` module and replaces the stub `hd-prompts-view` with a full split-panel layout composing the list and form.

## Files to Create

### 1. `app/client/ui/modules/prompt-form.ts`

New stateful module:
- `@property()`: `prompt: Prompt | null` — null = create mode, populated = edit mode
- `@state()`: `submitting: boolean`, `error: string`
- Form fields: name input, stage dropdown (disabled when `prompt` is set), instructions textarea (monospace), description textarea
- Submit handler: extracts values via `FormData`, calls `PromptService.create()` or `.update()` based on mode
- Dispatches `prompt-saved` (detail: `{ prompt }`) on success, `cancel` on cancel button click
- Shows error message on API failure
- Reuses `buttonStyles` from `@styles/buttons.module.css`
- Imports `PromptService` and `Prompt`/`CreatePromptCommand`/`UpdatePromptCommand` types

### 2. `app/client/ui/modules/prompt-form.module.css`

Form styling:
- Vertical flex layout with gap
- Form inputs matching existing toolbar input styling (border, radius, bg, mono font)
- Instructions textarea gets `font-family: var(--font-mono)`
- Error message styled with `--red` color
- Form actions row (save/cancel buttons) at bottom
- Consistent with existing design tokens

## Files to Modify

### 3. `app/client/ui/modules/index.ts`

Add `export { PromptForm } from "./prompt-form";`

### 4. `app/client/ui/views/prompts-view.ts`

Replace stub with full view composition:
- `@state()`: `selectedPrompt: Prompt | null`, `showForm: boolean`
- Split layout: left panel (`hd-prompt-list`), right panel (conditional `hd-prompt-form`)
- Event coordination:
  - `prompt-select` from list → `PromptService.find(id)` → set `selectedPrompt`, `showForm = true`
  - `create` from list → `selectedPrompt = null`, `showForm = true`
  - `prompt-saved` from form → `showForm = false`, refresh list, clear selection
  - `cancel` from form → `showForm = false`, clear selection
  - `prompt-deleted` from list → if deleted id matches `selectedPrompt.id`, clear form
- Pass `selectedId` to `hd-prompt-list` for highlight sync
- Import `PromptService` and `Prompt` type

### 5. `app/client/ui/views/prompts-view.module.css`

Replace stub CSS with split-panel layout:
- `:host` — flex column with padding (matches documents-view)
- `.view` — flex column container
- `.view-content` — flex row, split layout
- `.list-panel` — fixed/min width (~320-360px), overflow-y auto, border-right
- `.form-panel` — flex: 1, fills remaining space
- View header with title

## Event Flow

```
List "create" → View sets showForm=true, selectedPrompt=null
List "prompt-select" → View fetches prompt, sets showForm=true, selectedPrompt=prompt
List "prompt-deleted" → View clears form if deleted prompt was selected
Form "prompt-saved" → View hides form, refreshes list
Form "cancel" → View hides form, clears selection
```

## Patterns to Follow

- **View composition**: Same pattern as `documents-view.ts` — view manages UI state, coordinates modules via events + querySelector
- **FormData extraction**: Per web-dev skill convention, not controlled inputs
- **Stateless service calls**: Direct `PromptService.*` calls from modules
- **Flat UI convention**: Files at `app/client/ui/modules/prompt-form.ts` (not nested in subdirectory)
- **Import convention**: third-party → cross-package aliased → relative → styles

## Phase 5: Comprehensive Prompts-View Evaluation

Since this is the first time all prompts-view infrastructure is assembled together, Phase 5 is a comprehensive evaluation of the entire prompts view — not just the new form/view code, but also the list (#83) and card (#82) modules. Any issues discovered (visual, functional, UX) are fixed in-place before moving forward. The view must be in a finished state before closeout.

Evaluation scope:
- **Form**: create/edit modes, validation, error display, FormData extraction, submit/cancel flow
- **List**: search, filtering, sorting, pagination, activate/deactivate, delete confirmation
- **Card**: display, selection highlight, stage badge, active indicator, actions
- **View composition**: split-panel layout, event coordination, state sync between panels
- **CSS**: responsive behavior, token consistency, spacing, typography

## Build Verification

1. `bun run build` — clean build with no errors
2. Manual testing in browser at `/app/prompts`
