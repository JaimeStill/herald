# Plan: #82 — Prompt Card Pure Element and Search Request Type

## Context

First sub-issue of Objective #60 (Prompt Management View). This creates the foundational pure element for displaying prompt cards and a typed search request for server-side filtering. These are prerequisites for the list module (#83) and form/view (#84).

## Changes

### 1. Add `SearchRequest` type — `app/client/domains/prompts/prompt.ts`

Add a self-contained `SearchRequest` interface with all fields defined directly (no `PageRequest` extension). Follows the documents domain convention where each domain owns its own search request type.

```ts
export interface SearchRequest {
  page?: number;
  page_size?: number;
  search?: string;
  sort?: string;
  stage?: PromptStage;
  name?: string;
  active?: boolean;
}
```

### 2. Update `PromptService` param types — `app/client/domains/prompts/service.ts`

Migrate both `list()` and `search()` to use the new `SearchRequest` type (matching documents service pattern):
- `list(params?: PageRequest)` → `list(params?: SearchRequest)`
- `search(body: PageRequest)` → `search(body: SearchRequest)`
- Remove `PageRequest` from `@core` import, add `SearchRequest` to local `./prompt` import.

### 3. Update prompts barrel — `app/client/domains/prompts/index.ts`

Ensure `SearchRequest` is exported (already covered by `export * from "./prompt"`).

### 4. Add stage badge variants — `app/client/design/styles/badge.module.css`

Add `.classify`, `.enhance`, `.finalize` nested within `.badge`:
- `.classify` → blue (existing tokens)
- `.enhance` → orange (existing tokens)
- `.finalize` → green (existing tokens)

### 5. Create prompt card element — `app/client/ui/elements/prompts/prompt-card.ts`

New file following `hd-document-card` conventions:
- `@customElement("hd-prompt-card")`
- Properties: `prompt: Prompt`, `selected: boolean`
- Events: `select` (`{ id }`), `toggle-active` (`{ id, active }`), `delete` (`{ prompt }`)
- Renders: name, stage badge, active indicator, truncated description, delete button
- Imports shared badge + button styles, local CSS module
- Selected state via `.selected` class on card

### 6. Create prompt card CSS — `app/client/ui/elements/prompts/prompt-card.module.css`

Following document-card.module.css patterns:
- Card layout (flex column, gap, bg, border, radius)
- Selected state (blue border)
- Header with name + stage badge + active indicator
- Truncated description
- Actions row with toggle-active and delete buttons

### 7. Create prompts elements barrel — `app/client/ui/elements/prompts/index.ts`

Export `PromptCard` from `./prompt-card`.

### 8. Update elements barrel — `app/client/ui/elements/index.ts`

Add `export * from "./prompts"`.

## Files Modified

- `app/client/domains/prompts/prompt.ts` — add `PromptSearchRequest`
- `app/client/domains/prompts/service.ts` — update `search()` param type
- `app/client/design/styles/badge.module.css` — add stage variants
- `app/client/ui/elements/prompts/prompt-card.ts` — **new**
- `app/client/ui/elements/prompts/prompt-card.module.css` — **new**
- `app/client/ui/elements/prompts/index.ts` — **new**
- `app/client/ui/elements/index.ts` — add prompts export

## Verification

- `bun run build` succeeds with no errors
- Inspect built output includes the new element registration
- Stage badge classes render with correct colors (blue/orange/green)
