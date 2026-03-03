# 75 - Document Card and Classify Progress Elements

## Context

Issue #75 creates two pure elements for the documents view. Before building them, the web-development skill has a contradiction that must be resolved: `state.md`'s "Encapsulated Streaming" example shows `hd-document-card` calling `ClassificationService.classify()` directly — making it a stateful component, not a pure element. This violates the three-tier hierarchy and the anti-pattern "Pure elements calling services."

The fix: **streaming orchestration belongs in the stateful component tier**, not in pure elements. The card stays pure (props in, events out), and the list/grid component (#77) will own SSE lifecycle. The card emits a `classify` event; the parent handles it.

This session delivers the two elements AND updates the skill references to resolve the contradiction.

## Architecture Decisions

### Streaming Orchestration (skill correction)

The `state.md` "Encapsulated Streaming" example was placed in the wrong tier. The principle "component closest to the concern" is correct, but the card is closest to the **presentation** concern, not the streaming concern. The stateful list/grid component that renders cards is closest to the collection/streaming concern — it knows about all documents, can coordinate bulk operations, and is the natural bridge between services and elements.

**Updated pattern**: Stateful component calls `ClassificationService.classify()`, tracks per-document progress in `@state()` (e.g., `Map<string, progress>`), and passes progress data to pure cards as `@property()` values.

### Pure element import boundary

Pure elements import **only framework primitives** (`lit`, their own `*.module.css`). Nothing from the application — no domain services, no router utilities, no `@lit/context`. This is a hard boundary with no exceptions.

The cost is one extra event handler in the parent for navigation, but the payoff is significant:
- Elements are self-documenting: `@property()` shows inputs, `CustomEvent` shows outputs
- Elements are context-free: reusable in any parent (navigation, modal, embed)
- No judgment calls about what qualifies as a "utility" vs. a "service"
- Convention is universally applicable across any project

The card dispatches a `review` CustomEvent. The parent calls `navigate()`.

**Type imports are fine.** `import type { Document } from '@app/documents'` is erased at compile time — it creates zero runtime coupling. The boundary is **runtime imports**: only `lit` and the element's own CSS module. Type-only imports from domain modules are permitted because they describe the shape of data the element receives, without creating dependencies on application behavior.

### Directory structure

Three component-type directories under `app/client/`, each with domain subdirectories:

```
app/client/
├── views/           # route-level view components
│   └── documents/
├── components/      # stateful components (@consume, service calls)
│   └── (created in #77)
├── elements/        # pure elements (@property, CustomEvent)
│   └── documents/
│       ├── document-card.ts
│       ├── document-card.module.css
│       ├── classify-progress.ts
│       ├── classify-progress.module.css
│       └── index.ts
```

Elements are registered via side-effect import in `app.ts`: `import './elements'`.

## Implementation

### Step 1: `hd-classify-progress` element

**`app/client/elements/documents/classify-progress.ts`**

Pure element rendering a horizontal 4-stage pipeline: init → classify → enhance → finalize.

Properties:
- `currentNode: string` — the active stage
- `completedNodes: string[]` — stages that have finished

Stage state logic (hardcoded stage list `['init', 'classify', 'enhance', 'finalize']`):
- In `completedNodes` → `completed` class
- Equals `currentNode` → `active` class
- Otherwise → `pending` (default)

Renders as a flex row of stage indicators with connecting lines between them. Each stage: a small circle/dot + label below. Active stage gets a subtle pulse animation. Completed stages get the green accent. Pending stages are dimmed.

**`app/client/elements/documents/classify-progress.module.css`**

CSS using design tokens: `--green` for completed, `--blue` for active, `--color-2` for pending. Pulse keyframe for the active indicator. Flex layout with `gap` for spacing, connecting line segments via `::before`/`::after` pseudo-elements or border-based connectors.

### Step 2: `hd-document-card` element

**`app/client/elements/documents/document-card.ts`**

Pure element displaying document information with action buttons.

Properties:
- `document: Document` — document data (includes optional `classification`, `confidence`, `classified_at` from LEFT JOIN)
- `classifying: boolean` — whether classification SSE is in progress (default `false`)
- `currentNode: string` — current SSE stage (passed through to `hd-classify-progress`)
- `completedNodes: string[]` — completed SSE stages (passed through to `hd-classify-progress`)

Renders:
- **Header**: filename
- **Metadata**: page count, formatted upload date, file size
- **Status badge**: CSS class driven by `document.status` — `pending` (yellow tokens), `review` (blue tokens), `complete` (green tokens)
- **Classification summary** (conditional — when `document.classification` exists): classification label + confidence
- **Classify progress** (conditional — when `classifying` is true): `<hd-classify-progress>` with currentNode/completedNodes
- **Actions**: Classify button (dispatches `classify` CustomEvent), Review button (dispatches `review` CustomEvent)

Classify button disabled when `document.status === 'complete'` OR `classifying` is true.

Events dispatched:
- `classify` — `CustomEvent<{ id: string }>` with `bubbles: true, composed: true`
- `review` — `CustomEvent<{ id: string }>` with `bubbles: true, composed: true`

**`app/client/elements/documents/document-card.module.css`**

Card layout with `--bg-1` background, `--radius-md` border radius, `--shadow-sm` elevation. Status badge as inline pill using semantic color tokens. Action buttons in a footer row. Responsive-friendly with min-width constraints.

### Step 3: Barrel exports and registration

**`app/client/elements/documents/index.ts`**
```typescript
export { ClassifyProgress } from './classify-progress';
export { DocumentCard } from './document-card';
```

**`app/client/elements/index.ts`**
```typescript
export * from './documents';
```

**`app/client/app.ts`** — add `import './elements';` alongside existing `import './views';`

### Step 4: Update web-development skill references

**`references/state.md`** — Rewrite "Encapsulated Streaming" section:
- Show a stateful **list/grid component** (not the card) calling `ClassificationService.classify()`
- List component tracks `Map<string, { status, currentNode, completedNodes }>` via `@state()`
- List passes progress data to pure cards as properties
- List dispatches `classify-complete` to view for data refresh

**`references/components.md`** — Update:
- Stateful component example shows streaming orchestration
- Pure element example shows the card receiving classify progress as properties and dispatching classify event
- Pure element example dispatches `review` event instead of calling `navigate()` directly

**`SKILL.md`** — Update:
- Architecture Overview: clarify "components call services directly" means views and stateful components
- File Structure: add `elements/` and `components/` directories with domain subdirectories
- Naming Conventions: add element file naming pattern
- Anti-Patterns: strengthen "Pure elements calling services" — no application imports at all, including router
- Add convention: "Pure elements import only `lit` and their own CSS module — nothing from the application"

## Affected Files

New:
- `app/client/elements/documents/classify-progress.ts`
- `app/client/elements/documents/classify-progress.module.css`
- `app/client/elements/documents/document-card.ts`
- `app/client/elements/documents/document-card.module.css`
- `app/client/elements/documents/index.ts`
- `app/client/elements/index.ts`

Modified:
- `app/client/app.ts` — add elements import
- `.claude/skills/web-development/SKILL.md` — architecture updates
- `.claude/skills/web-development/references/state.md` — fix encapsulated streaming
- `.claude/skills/web-development/references/components.md` — update examples

## Validation Criteria

- [ ] `hd-classify-progress` renders 4-stage pipeline with pending/active/completed visual states
- [ ] `hd-document-card` renders filename, page count, date, size, status badge
- [ ] Status badge visually differentiates pending/review/complete via color tokens
- [ ] Classification summary shown conditionally when document has classification data
- [ ] Classify button dispatches `classify` CustomEvent with document ID
- [ ] Classify button disabled when status === 'complete' or classifying is true
- [ ] Review button dispatches `review` CustomEvent with document ID
- [ ] Progress element shown inline when classifying is true
- [ ] Neither element has runtime imports beyond `lit` and its own CSS module (`import type` from domain modules is OK)
- [ ] Both elements use `*.module.css` with design tokens
- [ ] `app.ts` imports `./elements` for registration
- [ ] Skill references updated — no more contradiction between tiers and streaming
- [ ] `bun run build` compiles without errors
