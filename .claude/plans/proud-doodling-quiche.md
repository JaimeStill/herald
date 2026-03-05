# 83 - Prompt List Module

## Context

Issue #83 is the second sub-issue of Objective #60 (Prompt Management View). It creates the `hd-prompt-list` stateful module that manages browsing, filtering, pagination, and prompt lifecycle actions. Depends on #82 (prompt card element), which is merged.

## Architecture Approach

Follow the `hd-document-grid` module pattern exactly. The prompt list is simpler — no SSE streaming, no bulk selection. It uses `PromptService.search()` (POST) for server-side filtering with `SearchRequest` body.

## Files to Create

### 1. `app/client/ui/modules/prompt-list.module.css`

Reuse the document-grid layout structure but use a vertical list instead of a grid (prompts are text-heavy per objective decision). Key differences:
- `.list` replaces `.grid` — single-column flex layout instead of CSS grid
- Toolbar identical to document-grid (search input, filter select, sort select, "New" button)
- Same empty state styling

### 2. `app/client/ui/modules/prompt-list.ts`

`@customElement("hd-prompt-list")` extending `LitElement`:

**State:**
- `@state() prompts: PageResult<Prompt> | null = null`
- `@state() page = 1`
- `@state() search = ""`
- `@state() stage = ""` (empty string = all stages)
- `@state() sort = "Name"` (alphabetical default)
- `@state() deletePrompt: Prompt | null = null`
- `@property({ type: String }) selectedId = ""`

**Private fields:**
- `searchTimer = 0`

**Lifecycle:**
- `connectedCallback()` → `fetchPrompts()`
- `disconnectedCallback()` → `clearTimeout(searchTimer)`

**Public method:**
- `refresh()` → reset page to 1, re-fetch

**Private methods:**
- `fetchPrompts()` → build `SearchRequest`, call `PromptService.search()`, set `this.prompts`
- `handleSearchInput()` → 300ms debounce, reset page
- `handleStageFilter()` → set stage, reset page, fetch
- `handleSort()` → set sort, reset page, fetch
- `handlePageChange()` → set page, fetch
- `handleSelect()` → dispatch `prompt-select` event with `{ id }`
- `handleToggleActive()` → call `PromptService.activate()` or `.deactivate()`, re-fetch
- `handleDelete()` → set `deletePrompt` state (shows confirm dialog)
- `confirmDelete()` → call `PromptService.delete()`, dispatch `prompt-deleted`, re-fetch
- `cancelDelete()` → clear `deletePrompt`
- `handleNew()` → dispatch `create` event

**Rendering:**
- `renderToolbar()` — search input, stage dropdown (All/classify/enhance/finalize), sort select, "New" button
- `renderList()` — map prompts to `<hd-prompt-card>` elements in a vertical list
- `render()` — toolbar + list + pagination + conditional confirm dialog

**Events dispatched:**
- `prompt-select` (`{ id }`) — card selected
- `create` — "New" button clicked
- `prompt-deleted` (`{ id }`) — after successful delete

## Files to Modify

### 3. `app/client/ui/modules/index.ts`

Add: `export { PromptList } from "./prompt-list";`

## Validation Criteria

- Module fetches and displays paginated prompts on mount
- Search input filters with 300ms debounce
- Stage dropdown filters by workflow stage
- Toggle active calls activate/deactivate API and refreshes list
- Delete shows confirmation dialog and removes prompt on confirm
- `prompt-select`, `create`, and `prompt-deleted` events dispatched correctly
- Pagination controls navigate pages
- Public `refresh()` method works
- Barrel exports updated
- `go vet ./...` passes
- Bun build succeeds
