# Objective Planning: #60 — Prompt Management View

## Context

Objective #60 decomposes the Prompt Management View into executable sub-issues. This is the fourth objective in Phase 3 (Web Client, v0.3.0). The previous objective (#59, Document Management View) is 100% complete (4/4 sub-issues closed) and needs transition closeout.

The prompt domain layer (types + stateless service) and backend API are fully implemented. A view stub exists at `/app/prompts` with placeholder content. The work is purely UI: building the components that let users browse, create, edit, delete, and toggle activation of named prompt overrides.

## Pre-Implementation: Web-Development Skill Update

The skill's `references/components.md` and `references/state.md` describe Signal.State + @lit/context as the primary pattern, but the actual codebase uses `@state()` decorators with direct service calls. The skill must be updated to align with reality before sub-issues are created.

### Changes to `SKILL.md`

- Update "Services and State" table: Shared state row should describe `@state()` in modules as the primary pattern, with Signal.State/context noted as available for genuine cross-subtree sharing
- Update three-tier table: View uses `@state()` and services; Module uses `@state()` and services; remove `@provide`/`@consume`/`SignalWatcher` as default tools
- Update "Prefer" list: `@state()` with direct service calls as primary; context/signals only when multiple descendants need the same reactive data
- Update triggers in frontmatter: remove `@provide`, `@consume`, `SignalWatcher` (these aren't used)

### Changes to `references/components.md`

- View example: Replace `SignalWatcher(LitElement)` + `Signal.State` + `@provide` with plain `LitElement` + `@state()` + direct service calls. Match actual `documents-view.ts` pattern (manages `showUpload` toggle, composes modules, relays events via querySelector).
- Module example: Replace `SignalWatcher(LitElement)` + `@consume` with plain `LitElement` + `@state()`. Match actual `document-grid.ts` pattern (owns fetch, search, filter, pagination, SSE orchestration as `@state()` fields).
- Pure element example: Already correct — `LitElement` with `@property()` and `CustomEvent`.

### Changes to `references/state.md`

- Reframe: `@state()` is the primary state management tool for both views and modules
- Context/signals section: Keep as an "available when needed" option, not the default
- Remove the elaborate Signal.State examples as the primary pattern
- Primary example: Module owns `@state()` fields, calls services, passes data to elements via `@property()`
- View coordination: Shows `querySelector` + `refresh()` pattern (as in documents-view.ts)

## Transition Closeout (Objective #59)

- #59 is 100% complete — all 4 sub-issues closed, no incomplete work
- Close #59 via `gh issue close 59`
- Update `_project/phase.md` — mark Objective 3 (Document Management View) as Complete
- Delete `_project/objective.md` and recreate for #60

## Architecture Decisions

1. **`@state()` + direct service calls**: Modules own their state via `@state()` and call services directly. No Signal.State, no @lit/context. Consistent with how the documents view works.

2. **Utilitarian design consistency**: Stay within Herald's existing design tokens and system fonts. Apply frontend-design thinking to layout composition and spatial quality within the existing system — clean proportions, thoughtful spacing, functional refinement.

3. **List layout, not grid**: Prompts are text-heavy (name, description, instructions). A vertical card list is more scannable than a responsive grid.

4. **`PromptSearchRequest` type**: The backend search supports `stage`, `name`, and `active` filters. Add a typed request to pass these through.

5. **Active toggle on card, not in form**: Activate/deactivate is atomic (auto-deactivates the previous active prompt for the stage). Quick-toggle on the card is more intuitive than a form checkbox.

6. **View fetches full prompt on select**: Avoids passing large instruction text through event details. `PromptService.find(id)` ensures the form gets complete data.

7. **Stage dropdown disabled in edit mode**: Changing stage of an existing prompt is semantically unusual and could cause unexpected deactivation.

## Sub-Issue Decomposition (3 sub-issues)

### Sub-Issue 1: Prompt card pure element and search type

**No dependencies.**

Scope:
- Add `PromptSearchRequest` interface to `app/client/domains/prompts/prompt.ts` (extends `PageRequest` with optional `stage`, `name`, `active`)
- Update `PromptService.search()` param type in `app/client/domains/prompts/service.ts`
- Export from `app/client/domains/prompts/index.ts`
- Create `hd-prompt-card` pure element at `app/client/ui/elements/prompts/`
  - `@property()` inputs: `prompt: Prompt`, `selected: boolean`
  - Events: `select` (`{ id }`), `toggle-active` (`{ id, active }`), `delete` (`{ prompt }`)
  - Renders: name, stage badge, active indicator, truncated description, delete button
- Add stage badge variants (`.classify`, `.enhance`, `.finalize`) to `app/client/design/styles/badge.module.css`
- Barrel exports: new `app/client/ui/elements/prompts/index.ts`, update `app/client/ui/elements/index.ts`

Files:
- CREATE `app/client/ui/elements/prompts/prompt-card.ts`
- CREATE `app/client/ui/elements/prompts/prompt-card.module.css`
- CREATE `app/client/ui/elements/prompts/index.ts`
- MODIFY `app/client/ui/elements/index.ts`
- MODIFY `app/client/domains/prompts/prompt.ts`
- MODIFY `app/client/domains/prompts/service.ts`
- MODIFY `app/client/domains/prompts/index.ts`
- MODIFY `app/client/design/styles/badge.module.css`

### Sub-Issue 2: Prompt list module

**Depends on Sub-Issue 1.**

Scope:
- Create `hd-prompt-list` stateful module at `app/client/ui/modules/prompts/`
  - `@state()`: `prompts`, `page`, `search`, `stage`, `sort`, `deletePrompt`
  - `@property()`: `selectedId` (from view, for highlighting)
  - Toolbar: search input (300ms debounce), stage filter dropdown, "New" button
  - Renders `hd-prompt-card` elements in a vertical list
  - Handles: `toggle-active` (calls activate/deactivate), `delete` (confirm dialog)
  - Dispatches: `prompt-select` (`{ id }`), `create`, `prompt-deleted` (`{ id }`)
  - Pagination via `hd-pagination`
  - Public `refresh()` method
- Barrel exports: new `app/client/ui/modules/prompts/index.ts`, update `app/client/ui/modules/index.ts`

Files:
- CREATE `app/client/ui/modules/prompts/prompt-list.ts`
- CREATE `app/client/ui/modules/prompts/prompt-list.module.css`
- CREATE `app/client/ui/modules/prompts/index.ts`
- MODIFY `app/client/ui/modules/index.ts`

Reference: `app/client/ui/modules/documents/document-grid.ts` (toolbar, search debounce, pagination, confirm dialog pattern)

### Sub-Issue 3: Prompt form module and view integration

**Depends on Sub-Issues 1 and 2.**

Scope:
- Create `hd-prompt-form` stateful module at `app/client/ui/modules/prompts/`
  - `@property()`: `prompt: Prompt | null` (null = create mode)
  - `@state()`: `submitting`, `error`
  - FormData extraction on submit (per skill convention — not controlled inputs)
  - Form: name input, stage dropdown (disabled in edit), instructions textarea (monospace), description textarea
  - Calls `PromptService.create()` or `.update()`
  - Dispatches: `prompt-saved` (`{ prompt }`), `cancel`
- Update `hd-prompts-view` to compose list + form with split layout
  - `@state()`: `selectedPrompt`, `showForm`
  - Left panel: `hd-prompt-list`, right panel (conditional): `hd-prompt-form`
  - Coordinates: select → find → populate form, save → refresh list, delete → clear form if selected
- Update barrel export in `app/client/ui/modules/prompts/index.ts`

Files:
- CREATE `app/client/ui/modules/prompts/prompt-form.ts`
- CREATE `app/client/ui/modules/prompts/prompt-form.module.css`
- MODIFY `app/client/ui/modules/prompts/index.ts`
- MODIFY `app/client/ui/views/prompts/prompts-view.ts`
- MODIFY `app/client/ui/views/prompts/prompts-view.module.css`

Reference: `app/client/ui/views/documents/documents-view.ts` (view composition, querySelector coordination)

## Dependency Graph

```
Sub-Issue 1 (Card + Search Type)
      |
      v
Sub-Issue 2 (List Module)
      |
      v
Sub-Issue 3 (Form + View Assembly)
```

## Key Reference Files

- `app/client/ui/modules/documents/document-grid.ts` — stateful module pattern
- `app/client/ui/elements/documents/document-card.ts` — pure element pattern
- `app/client/ui/views/documents/documents-view.ts` — view composition pattern
- `app/client/domains/prompts/service.ts` — service API (already complete)
- `app/client/domains/prompts/prompt.ts` — types (needs PromptSearchRequest)
- `app/client/design/styles/badge.module.css` — shared badge styles (needs stage variants)

## Skill Files to Update (pre-implementation)

- `.claude/skills/web-development/SKILL.md` — realign architecture overview with codebase
- `.claude/skills/web-development/references/components.md` — replace Signal.State examples with @state() pattern
- `.claude/skills/web-development/references/state.md` — reframe @state() as primary, context/signals as optional
