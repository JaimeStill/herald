# B3b — `core/upload.typ` + checkpoint-2

## Context

The Herald OV-1 briefing session is mid-flight. B3a (`core/readme.typ`) landed after 16 design iterations and a long context burn. The user has now flagged that diagram authoring is heavy enough that the session should **checkpoint after every diagram**, not in big batches, so context stays bounded between resumes.

This plan scopes a single executable unit: author `core/upload.typ`, render it, iterate to acceptance, then write `checkpoint-2.md`. Once the user accepts the rendered diagram and the checkpoint lands, the session pauses; B3c–B3e + B4–B7 resume in fresh contexts.

The diagram itself is the second of five OV-1 stakeholder-voiced diagrams. The technical-writer subagent's findings (`~/code/herald/.claude/context/sessions/2026-04-29-herald-ov1/findings.md`, *upload* section) lock the entities, relationships, layout intent, and visual approach. Open question: **edge-label wording** (lowercase verb phrases, stakeholder voice) and **return-edge styling** (dashed + bend-below, per the edges-and-marks reference).

## Inputs already gathered (no re-exploration needed)

| Source | What it gives |
|---|---|
| `~/code/herald/.claude/context/sessions/2026-04-29-herald-ov1/checkpoint-1.md` | B3a conventions: shape-body-as-semantic, edges-as-connectors-only, two-node container, cardinal anchors, label-fill pattern. Adaptation note: **flow diagrams retain edge labels** when relationships are step names (this diagram). |
| `~/code/herald/.claude/context/sessions/2026-04-29-herald-ov1/findings.md` | Per-diagram entities, relationships, layout intent, prose draft. Sensitivity check passed. |
| `~/code/herald/diagrams/core/readme.typ` | The `kinded(pos, hue, glyph, title, kind, description, extras: none, ref: none)` helper — copy-paste into `upload.typ`. |
| `~/code/herald/diagrams/design/{tokens,theme}.typ` | Tokens (`gap-cell`, `gap-structured-text`, `pad-inside-shape`, `space-between-shapes`, `label-sep: 10pt`, `label-size: 9.5pt`, `size-body: 11pt`, `stroke-thin/default/emphasis`), hue families (blue, coral, orange, green, …), neutral palette. |
| `~/tau/diagrams/.claude/skills/diagram-ingredients/references/labels-and-encapsulation.md` | Native `label-fill: palette.surface` + `label-pos` + `label-side` (don't re-use the custom `box` wrapper from checkpoint-1; Fletcher's native `label-fill` is canonical). |
| `~/tau/diagrams/.claude/skills/diagram-ingredients/references/edges-and-marks.md` | Mark vocabulary, dashed-style body (`--`), waypoints (explicit-tuple form), return-edge pattern (`bend: -25deg` with `stroke: (paint: ..., dash: "dashed")`). |
| `~/code/herald/diagrams/mise.toml` | Render command: `mise run render-file core/upload.typ` → `core/upload-{light,dark}.svg`. |
| `~/tau/diagrams/.claude/skills/diagram-authoring/references/critique-checklist.md` | Concept/audience, identity/metadata, visual weight, edges/routing, containers, color/theme, typography, render hygiene, source materials, sketch alignment, remediation. |

**Toolkit precedent for step-and-actor flow:** none in TAU's rendered corpus. This diagram surfaces the convention. Closest reference is `~/tau/diagrams/ingredients/edges-and-marks/parallel-edges.typ` for the bend + label-fill pattern.

## Design

### Entities, hues, glyphs

| Ref | Entity | Kind | Hue | Glyph | Description |
|---|---|---|---|---|---|
| `<client>` | Upload client | upstream | `palette.orange` | `\u{F007}` (user) | Hands documents to Herald for classification. |
| `<h>` | Herald | service | `palette.blue` | `\u{F0AC}` (globe) | Receives uploads; durably stores each document and registers it for classification. |
| `<blob>` | Azure Blob Storage | object storage | `palette.coral` | `\u{F0A0}` (hard-drive) | Durable, immutable document store. |
| `<pg>` | Azure PostgreSQL | database | `palette.coral` | `\u{F1C0}` (database) | Document registration record. |

Hue and glyph choices echo `readme.typ` for cross-diagram cohesion (per findings open-question Q3 — shared visual vocabulary). Descriptions are intentionally lighter than `readme.typ`'s — flow shapes carry actor identity, not dependency context.

### Layout (coords)

```
   <client>(-2, 0) ─────▶ <h>(0, 0) ─────▶ <blob>(2, -0.5)
                              │
                              └────────▶ <pg>(2, 0.5)
                              ◀── (return, dashed, bend below)
```

- Diagram `spacing` matches `readme.typ`: `(3 * tokens.space-between-shapes, 2.5 * tokens.space-between-shapes)`.
- `<blob>` and `<pg>` stacked at `x = 2`, `y = ±0.5` — vertical fan-out from Herald reads as parallel writes without crossing.
- Return edge runs from `<h.south-west>` to `<client.south-east>` with `bend: -25deg` so it curves *below* the forward edge.

### Edges (labels retained — flow handoffs)

| Edge | Mark | Label | Notes |
|---|---|---|---|
| `<client.east> → <h.west>` | `->` | `upload document` | Forward. Single straight handoff. |
| `<h.east> → <blob.west>` | `->` | `store immutably` | Forward. Top branch of fan-out. |
| `<h.east> → <pg.west>` | `->` | `register record` | Forward. Bottom branch of fan-out. |
| `<h.south-west> → <client.south-east>` | `-->` (dashed) | `return identifier` | Return. `bend: -25deg`. `stroke: (paint: palette.green.stroke, dash: "dashed")`. |

**Edge stroke** matches `readme.typ` baseline: `tokens.stroke-default + palette.green.stroke` for the three forward edges; the return edge swaps to a dashed paint via `stroke: (paint: ..., dash: "dashed")`.

**Label styling (Fletcher-native, per labels-and-encapsulation reference):**
```typst
edge(
  <client.east>, <h.west>, "->",
  text(size: tokens.size-label, weight: tokens.weight-light,
       fill: palette.ink-muted, style: "italic", "upload document"),
  label-pos: 0.5,
  label-side: center,
  label-sep: tokens.label-sep,
  label-fill: palette.surface,
  stroke: edge-stroke,
)
```

Note: `checkpoint-1.md` §6 referenced a custom `lbl()` `box` helper. The labels-and-encapsulation reference now favours Fletcher's native `label-fill` — adopt the native form (the `box` helper was a fallback before native support landed).

### Helper reuse

Copy `kinded` from `readme.typ` into `upload.typ` self-contained. **Defer extraction to a shared `_helpers.typ`** until B3e — once all five diagrams exist, evaluate whether extraction is worth the import-graph change. Premature extraction risks shape signatures that don't fit the later diagrams.

## File to write

**`~/code/herald/diagrams/core/upload.typ`** — single new file. Approximate size: ~120 lines (smaller than `readme.typ` because no container, fewer entities, no header-foreground / background-container split).

Skeleton:

```typst
// core/upload — How a document enters Herald (stakeholder voice).
//
// Step-and-actor flow: actors on a baseline, edges carry handoff labels.
// The actor handoff IS the concept (per findings). Three forward edges
// fan out from Herald to its two stores; a dashed return edge runs
// beneath the forward flow.

#import "@preview/fletcher:0.5.8" as fletcher: diagram, node, edge
#import "../design/tokens.typ": tokens
#import "../design/theme.typ": palette

#set page(width: auto, height: auto, margin: tokens.pad-inside-container, fill: palette.surface)
#set text(font: tokens.font, fill: palette.ink)

// Copy of `kinded` helper from readme.typ. Extract to shared module
// after B3e once the helper's signature is validated across all five
// diagrams.
#let kinded(pos, hue, glyph, title, kind, description, extras: none, ref: none) = node(...)

#let edge-stroke = tokens.stroke-default + palette.green.stroke
#let edge-stroke-dashed = (paint: palette.green.stroke, thickness: tokens.stroke-default,
                          dash: "dashed")

#let step-label(s) = text(size: tokens.size-label, weight: tokens.weight-light,
                          fill: palette.ink-muted, style: "italic", s)

#diagram(
  spacing: (3 * tokens.space-between-shapes, 2.5 * tokens.space-between-shapes),

  kinded((-2, 0), palette.orange, "\u{F007}",
    "Upload client", "upstream",
    [Hands documents to Herald \
     for classification.],
    ref: <client>),

  kinded((0, 0), palette.blue, "\u{F0AC}",
    "Herald", "service",
    [Receives uploads; durably stores \
     each document and registers it \
     for classification.],
    ref: <h>),

  kinded((2, -0.5), palette.coral, "\u{F0A0}",
    "Azure Blob Storage", "object storage",
    [Durable, immutable document store.],
    ref: <blob>),

  kinded((2, 0.5), palette.coral, "\u{F1C0}",
    "Azure PostgreSQL", "database",
    [Document registration record.],
    ref: <pg>),

  edge(<client.east>, <h.west>, "->",
    step-label("upload document"),
    label-pos: 0.5, label-side: center,
    label-sep: tokens.label-sep, label-fill: palette.surface,
    stroke: edge-stroke),

  edge(<h.east>, <blob.west>, "->",
    step-label("store immutably"),
    label-pos: 0.5, label-side: center,
    label-sep: tokens.label-sep, label-fill: palette.surface,
    stroke: edge-stroke),

  edge(<h.east>, <pg.west>, "->",
    step-label("register record"),
    label-pos: 0.5, label-side: center,
    label-sep: tokens.label-sep, label-fill: palette.surface,
    stroke: edge-stroke),

  edge(<h.south-west>, <client.south-east>, "-->",
    step-label("return identifier"),
    bend: -25deg,
    label-pos: 0.5, label-side: center,
    label-sep: tokens.label-sep, label-fill: palette.surface,
    stroke: edge-stroke-dashed),
)
```

## Execution order

1. **Author** `~/code/herald/diagrams/core/upload.typ` from the skeleton above.
2. **Render**: `cd ~/code/herald/diagrams && mise run render-file core/upload.typ`. Verify both `core/upload-light.svg` and `core/upload-dark.svg` produce.
3. **Critique** against the checklist: one concept (the upload handoff), shape-for-identity / text-for-metadata, visual weight (Herald heaviest), edges (labels readable, fan-out doesn't crowd, return doesn't fight forward), both themes render.
4. **Iterate** with the user. Likely friction points:
   - Fan-out edges from `<h.east>` may overlap labels at the midpoint — adjust `label-pos` to `0.4` / `0.6` if the two labels collide, or stagger Blob/PG y-coords further (`y = ±0.6`).
   - Return-edge bend might clip shape bodies — tune `bend` between `-20deg` and `-30deg`.
   - Edge-label wording — user may prefer alternative verb phrases; defer wording finalisation to the iteration step.
5. **Lock** when the user accepts.
6. **Write checkpoint-2** at `~/code/herald/.claude/context/sessions/2026-04-29-herald-ov1/checkpoint-2.md` (see template below).
7. **Pause session.** Hand back to user; B3c resumes in fresh context.

## Checkpoint-2 contents (template)

Mirror `checkpoint-1.md` structure. Status table updated: B3b ✓; B3c–B3e pending. Sections to include:

- **What landed in B3b** — final form of `upload.typ`, layout decisions, any iteration notes worth preserving.
- **Conventions surfaced or confirmed in B3b** — at minimum:
  - Step-and-actor flow with retained edge labels (vs. `readme.typ`'s edges-as-connectors).
  - Native Fletcher `label-fill` (supersedes the `lbl()` box helper from checkpoint-1 §6).
  - Vertical fan-out from a focal node to parallel stores (Herald → Blob + PG).
  - Dashed + bent-below return edge for response semantics.
  - `kinded` helper copy-paste rather than shared module (revisit at B3e).
- **Adaptation notes for B3c–B3e** — refine the table from checkpoint-1 §*Adaptation notes* with what was confirmed/changed during B3b.
- **Resume protocol** — same 7-step read order as checkpoint-1, swapping in checkpoint-2.
- **Pending toolkit updates** — propagate any new flow conventions worth back-applying to TAU's `diagram-ingredients/` references after the session.
- **Critical files** — same table, with `core/upload.typ` and `upload-{light,dark}.svg` added.
- **Open thread** — anything unresolved from B3b iterations.

## Verification

- `mise run render-file core/upload.typ` exits 0; produces both light and dark SVGs in `core/`.
- Manual visual inspection in both themes (open the SVGs).
- Critique checklist run against the rendered diagram.
- User explicit acceptance before writing checkpoint-2.
- Checkpoint-2 readable end-to-end as a self-contained resume artifact (test: read it cold and confirm the next session can act on it).

## Critical files

| Path | Role |
|---|---|
| `~/code/herald/diagrams/core/upload.typ` | **New** — B3b artifact. |
| `~/code/herald/diagrams/core/upload-{light,dark}.svg` | **New** — rendered output. |
| `~/code/herald/diagrams/core/readme.typ` | Source of `kinded` helper to copy. Read-only this session. |
| `~/code/herald/diagrams/design/{tokens,theme}.typ` | Token + palette imports. Read-only. |
| `~/code/herald/diagrams/mise.toml` | Render command. Read-only. |
| `~/code/herald/.claude/context/sessions/2026-04-29-herald-ov1/checkpoint-2.md` | **New** — session pause point. |
| `~/code/herald/.claude/context/sessions/2026-04-29-herald-ov1/findings.md` | Reference: upload entities/relationships/prose. Read-only. |
| `~/code/herald/.claude/context/sessions/2026-04-29-herald-ov1/checkpoint-1.md` | Reference: B3a conventions. Read-only. |
| `~/tau/herald-overview-brief.md` | Reference: locked decisions, B3a addendum. Read-only. |

## Out of scope for this plan

- Authoring B3c (`classification.typ`), B3d (`release.typ`), B3e (`cds-release.typ`).
- B4 (README assembly) — prose drafts already exist in findings; they are not written into `README.md` until B4.
- B5 (PDF render), B6 (phase-04 prelude), B7 (commit).
- Toolkit edits at `~/tau/diagrams/` — capture in checkpoint-2 *Pending toolkit updates*; apply in a separate session.
- Extracting `kinded` to a shared module — defer to B3e.
- Updating the brief addendum — do at B3e or B6, after the full set of conventions has settled.
