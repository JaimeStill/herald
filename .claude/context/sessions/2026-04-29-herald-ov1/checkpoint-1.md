# Session checkpoint — 2026-04-29 — Herald OV-1, after B3a

Resume point. Read this first when resuming the session in a fresh context.

## Status

| Phase | Step | Status |
|---|---|---|
| A | Revise brief | ✓ complete |
| B1 | Workspace setup | ✓ complete |
| B2 | Technical-writer findings | ✓ complete |
| B3a | `core/readme.typ` | ✓ complete |
| B3b | `core/upload.typ` | pending |
| B3c | `core/classification.typ` | pending |
| B3d | `core/release.typ` | pending |
| B3e | `core/cds-release.typ` | pending |
| B4 | Assemble README.md | pending |
| B5 | Render briefing PDF | pending |
| B6 | Phase-04 prelude artifact | pending |
| B7 | Commit | pending |

## What landed in B3a

`~/code/herald/diagrams/core/readme.typ` — final form, 16 iterations of refinement with the user. Renders to `core/readme-{light,dark}.svg`.

**Visual approach**: hub-and-spoke architecture, *compose fresh* per the technical-writer's recommendation. Two structural refactors during the session:

1. **Iteration 8** — *shape-body-as-semantic*. Edge labels migrated into shape descriptions; edges became connectors-only. Driven by edge/label thrashing in dense layouts (5+ entities, 6+ relationships, multi-line labels colliding with each other and shape bodies).
2. **Iteration 11** — *two-node container split*. The `enclose:` container's body content (header) rendered behind the inner shapes when `snap: -1` was set. Solution: split into a foreground header node + a background `enclose:` container with `snap: -1` and no body.

## Conventions discovered during B3a

These propagate to B3b–B3e (with adaptation for flow/process diagrams; see *Adaptation notes* below).

### 1. Shape body as semantic carrier

For dense diagrams (many components, many relationships), edge labels become unmanageable. Move the *what* of each relationship into the shape body; edges become connectors only. The reusable helper:

```typst
#let kinded(pos, hue, glyph, title, kind, description, extras: none, ref: none) = node(pos,
  grid(columns: (auto, auto), column-gutter: tokens.gap-cell,
    align: (left + top, left + top),
    text(size: 18pt, fill: hue.stroke, glyph),
    stack(dir: ttb, spacing: tokens.gap-structured-text,
      block(width: 100%, align(center,
        text(size: tokens.size-body, weight: tokens.weight-bold, fill: palette.ink, title))),
      block(width: 100%, align(center,
        text(size: tokens.size-label, weight: tokens.weight-light, fill: hue.ink,
          style: "italic", "(" + kind + ")"))),
      block(spacing: 0pt, line(length: 100%, stroke: tokens.stroke-thin + hue.divider)),
      text(size: tokens.size-label, fill: palette.ink, description),
      ..(if extras != none { (extras,) } else { () }),
    ),
  ),
  shape: fletcher.shapes.rect,
  fill: hue.fill,
  stroke: tokens.stroke-default + hue.stroke,
  inset: tokens.pad-inside-shape,
  corner-radius: tokens.radius-shape,
  name: ref,
)
```

Key details:
- Title and (kind) **centred** within stack natural width via `block(width: 100%, align(center, ...))`. `align(center, ...)` alone doesn't expand, only centres within content's natural width.
- Description **left-aligned** (default).
- Hue-aware divider (`hue.divider` is the design system's tonal mid-point between fill and stroke).
- Description text uses `palette.ink` (full contrast), not `palette.ink-muted`.
- Description content is markup `[...]` so `\` line breaks render correctly.

### 2. Edges-as-connectors-only

When shape bodies carry the relationship semantics, edges drop their labels. Direction is enough; shapes explain the relationship. Mark style differentiates relationship kinds:

| Mark | Reads as | Used for |
|---|---|---|
| `->` | data flow / dependency | standard |
| `-O` | association / dock | deployment-target (Herald → Container Apps) |
| `<->` | bidirectional cycle | Reviewer ↔ Herald validate cycle |

### 3. Two-node container pattern (header above inner shapes)

When a container needs a header that renders **above** the inner shapes (not behind them, as Fletcher's body-content does with `snap: -1`):

```typst
// Header as SEPARATE node, foreground (default layer)
node((header-x, header-y),
  stack(dir: ttb, ... header content ...),
  shape: fletcher.shapes.rect,
  fill: none, stroke: none, inset: 0pt,
  name: <header>,
)

// Background container — fill only, snap: -1, encloses header + inner nodes
node(
  enclose: (<header>, <inner-1>, <inner-2>, ...),
  [],  // empty body
  shape: fletcher.shapes.rect,
  fill: palette.surface-muted,
  stroke: none,                       // no border, just fill
  inset: tokens.pad-inside-container,
  corner-radius: tokens.radius-container,
  snap: -1,                           // behind everything
)
```

The container's bounding box spans the union of all enclosed nodes (header + inner). Header sits at default layer; container fill at `snap: -1`. Result: header text renders *on top* of the container fill but *next to* (not behind) the inner shapes — provided the header's coord positions it spatially above them.

### 4. Mark distinction for deployment / dock relationships

`-O` (large open circle terminator) reads as "association / dock" rather than "data flow". Used at the destination end of the deployment edge (Herald → Container Apps). Avoids the visual ambiguity of converging arrowheads at the focal node's perimeter.

Pair with `<aca.north>` as destination so the circle lands centred on the target's top edge.

### 5. Cardinal anchors trump fractional waypoints

Fletcher's auto-clip behaviour with fractional-coord waypoints is unreliable for precise edge endpoints. Use cardinal anchors (`<node.north>`, `<node.south>`, `<node.east>`, `<node.west>`, plus the four corners) when the connection point matters. Fractional coords inside a shape get clipped at the shape boundary anyway.

The 1/3 / 2/3 fractional alignment we tried in early iterations did not survive content-volatility — it depended on Herald's natural width, which moved as the description grew.

### 6. Label padding + `label-side: center` (kept for any future labelled edges)

If a future edge does need a label:

```typst
#let lbl(s) = box(
  inset: (x: 6pt, y: 2pt),
  text(size: tokens.label-size, weight: tokens.weight-light, fill: palette.ink-muted, style: "italic", s),
)

edge(..., lbl("..."), label-pos: 0.5, label-side: center, label-fill: palette.surface, ...)
```

The padded box gives `label-fill` visible breathing room; `label-side: center` puts the label *on* the line with the line breaking *through* the padded box.

## Adaptation notes for B3b–B3e

The 5 diagrams in this OV-1 are not all the same shape:

| File | Visual approach (per technical-writer findings) | Convention adaptation |
|---|---|---|
| `core/readme.typ` | hub-and-spoke (compose fresh) — **done** | shape-body-as-semantic + container split |
| `core/upload.typ` | step-and-actor flow | actors carry descriptions; **steps** likely retain edge labels (handoff names matter) |
| `core/classification.typ` | state-graph card with fork-and-rejoin | states carry descriptions; transition labels likely needed for the conditional fork |
| `core/release.typ` | labelled milestone strip | milestone shapes carry brief role; transitions may carry trigger names |
| `core/cds-release.typ` | labelled milestone strip with focal step | same; the cross-domain transfer is the focal step (heavier visual weight) |

The shape-body-as-semantic convention works well when relationships are CONTEXTUAL ("X depends on Y"). For flow diagrams where relationships are STEP NAMES ("X transitions to Y by doing Z"), edge labels often retain meaning. **Judge per diagram.**

## Resume protocol

1. Read this checkpoint.
2. Read `~/code/herald/.claude/plans/initialize-tau-herald-overview-brief-md-reactive-crane.md` — the original session plan.
3. Read `~/tau/herald-overview-brief.md` — revised brief; locked decisions and high-level conventions live here.
4. Read `~/code/herald/.claude/context/sessions/2026-04-29-herald-ov1/findings.md` — technical-writer per-diagram findings (entities, relationships, layout intent, prose drafts, sensitivity check).
5. Read `~/code/herald/diagrams/core/readme.typ` — internalise the `kinded` helper and the two-node container pattern for reuse.
6. Skim `~/tau/diagrams/.claude/skills/diagram-ingredients/references/labels-and-encapsulation.md` (updated this session — see toolkit deltas below).
7. Confirm with the user before resuming.

## Toolkit deltas applied this session

- **`~/tau/diagrams/.claude/skills/diagram-ingredients/references/edges-and-marks.md`** — Waypoints section rewritten to recommend explicit-tuple form; relative-direction-string form documented as unreliable in 0.5.x.
- **`~/tau/diagrams/ingredients/edges-and-marks/waypoints.typ`** — Catalog example switched to explicit-tuple form.
- **`~/tau/diagrams/.claude/skills/typst-diagrams/references/fletcher-pitfalls.md`** — New entry: comma-separated route strings parse as separate arguments (label-mis-interpretation + adjacent-vertex assertion).

Toolkit deltas pending (post-checkpoint, see *Pending toolkit updates* below).

## Pending toolkit updates

These should be applied next (separate, focused edits — small file changes):

1. **`~/tau/diagrams/.claude/skills/diagram-ingredients/references/labels-and-encapsulation.md`** — Add the two-node container variant for header-above-inner-nodes.
2. **`~/tau/diagrams/.claude/skills/diagram-authoring/SKILL.md`** or new reference — The "shape-body-as-semantic" pattern: when to choose body-borne semantics vs edge labels in dense diagrams.
3. **`~/tau/herald-overview-brief.md`** — Brief addendum: conventions established during B3a, for the four remaining diagrams.

## Critical files

| Path | Role |
|---|---|
| `~/code/herald/diagrams/core/readme.typ` | Final B3a artifact; reusable shape helpers (`kinded`) |
| `~/code/herald/diagrams/core/readme-{light,dark}.svg` | Rendered B3a |
| `~/code/herald/diagrams/{mise.toml,design/}` | Render pipeline + design layer mirrored from TAU |
| `~/code/herald/.claude/agents/technical-writer.md` | Subagent symlink → `~/tau/diagrams/.claude/agents/technical-writer.md` |
| `~/code/herald/.claude/context/sessions/2026-04-29-herald-ov1/findings.md` | Per-diagram structured findings |
| `~/code/herald/.claude/context/sessions/2026-04-29-herald-ov1/checkpoint-1.md` | This file |
| `~/code/herald/.claude/plans/initialize-tau-herald-overview-brief-md-reactive-crane.md` | Original session plan |
| `~/tau/herald-overview-brief.md` | Revised brief |
| `~/tau/diagrams/.claude/project/04-advanced-tau-diagrams.md` | Phase-04 plan |
| `~/tau/diagrams/.claude/agents/technical-writer.md` | Canonical subagent contract |
| `~/tau/diagrams/.claude/skills/tau-diagrams/SKILL.md` | TAU diagram standard |
| `~/tau/diagrams/.claude/skills/diagram-ingredients/references/labels-and-encapsulation.md` | Container pattern reference |
| `~/tau/diagrams/.claude/skills/diagram-ingredients/references/edges-and-marks.md` | Edges + marks (waypoints fixed this session) |
| `~/tau/diagrams/.claude/skills/typst-diagrams/references/fletcher-pitfalls.md` | Fletcher pitfalls (route-string parsing fixed this session) |

## Open thread

Two B3-related Fletcher behaviours we couldn't conclusively bypass:

1. **Mark overlay on shape**. Tried literal coord destination + `layer: 1`; both produced clipped arrows. Workaround was to switch the mark style (`-O`) so the visual sits naturally at the boundary. If a future diagram needs an arrow tip *inside* a shape, this remains unsolved.

2. **Header-vs-shape rendering layer in containers**. The two-node container pattern works because header and inner shapes don't spatially overlap. If a future container needs a header that overlaps the inner shapes vertically (e.g., compact dense container), this will need re-investigation.

Note both for the phase-04 prelude artifact.
