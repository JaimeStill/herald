# Issue #89 — Markings List Element and Classification Panel Module

## Context

Sub-issue 2 of Objective 5: Document Review View (#61). Creates two UI components: a pure element for rendering security marking badges, and a stateful module for displaying/interacting with classification data (validate and update actions). These will be composed into the review view in issue #90.

## Files to Create

1. `app/client/ui/elements/markings-list.ts` — pure element
2. `app/client/ui/elements/markings-list.module.css` — element styles
3. `app/client/ui/modules/classification-panel.ts` — stateful module
4. `app/client/ui/modules/classification-panel.module.css` — module styles

## Files to Modify

5. `app/client/ui/elements/index.ts` — add `MarkingsList` export
6. `app/client/ui/modules/index.ts` — add `ClassificationPanel` export

## Implementation

### Step 1: `hd-markings-list` Element

Pure element receiving `markings: string[]`. Renders each marking as a badge tag. Empty state when array is empty/undefined.

- `@property({ type: Array }) markings: string[] = []`
- Import `badgeStyles` from `@styles/badge.module.css`
- Each marking rendered as `<span class="badge">${marking}</span>`
- Markings badges use a neutral/marking-specific style (not tied to workflow stage colors)
- No events dispatched — display only

CSS: Flex-wrap container for badges, gap spacing, empty state message.

### Step 2: `hd-classification-panel` Module

Stateful module following `prompt-form.ts` patterns.

**Props:**
- `@property() documentId: string` — used to load classification
- `@property({ type: Object }) document: Document | null` — document context from parent

**State:**
- `@state() classification: Classification | null = null`
- `@state() loading = true`
- `@state() error = ""`
- `@state() mode: "view" | "validate" | "update" = "view"`
- `@state() submitting = false`

**Lifecycle:**
- `updated()` watches `documentId` changes, calls `ClassificationService.findByDocument()`
- On success: sets classification, clears loading
- On error: sets error, shows empty state with "Back to Documents" link

**Display (mode = "view"):**
- Classification value + confidence badge (reuse badge styles with confidence class)
- `<hd-markings-list>` with `classification.markings_found`
- Rationale in preformatted text
- Model/provider metadata row
- Timestamps (classified_at, validated_at) using `formatDate()`
- Validated-by (if present)
- Action buttons: "Validate" and "Update" to switch modes

**Validate mode:**
- Name input for `validated_by`
- "Validate" submit + "Cancel" button
- On submit: `ClassificationService.validate(id, { validated_by })` → update `@state()` from response → dispatch `validate` event → return to view mode

**Update mode:**
- Form fields: classification (text input), rationale (textarea), updated_by (text input)
- "Update" submit + "Cancel" button
- On submit: `ClassificationService.update(id, { classification, rationale, updated_by })` → update `@state()` from response → dispatch `update` event → return to view mode

**Key services/types used:**
- `ClassificationService` from `@domains/classifications` — `findByDocument`, `validate`, `update`
- `Classification` type from `@domains/classifications`
- `ValidateCommand`, `UpdateCommand` from `@domains/classifications`
- `formatDate` from `@core`
- `Document` type from `@domains/documents`

### Step 3: Barrel Updates

Add exports to element and module index files.

## Verification

- `bun run build` succeeds from `app/`
- Visual review: classification panel renders all sections
- Validate flow: enter name → submit → state updates, event dispatches
- Update flow: expand form → fill fields → submit → state updates, event dispatches
- Empty state: shows message when no classification exists
