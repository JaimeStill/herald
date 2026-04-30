# Session checkpoint — 2026-04-29 — Herald OV-1, after B3b

Resume point. Read this first when resuming the session in a fresh context. This supersedes `checkpoint-1.md` for resumption purposes; checkpoint-1 remains the canonical record of B3a's conventions and is referenced from this file.

## Status

| Phase | Step | Status |
|---|---|---|
| A | Revise brief | ✓ complete |
| B1 | Workspace setup | ✓ complete |
| B2 | Technical-writer findings | ✓ complete |
| B3a | `core/readme.typ` | ✓ complete |
| B3b | `core/upload.typ` | ✓ complete |
| B3c | `core/classification.typ` | pending |
| B3d | `core/release.typ` | pending |
| B3e | `core/cds-release.typ` | pending |
| B4 | Assemble README.md | pending |
| B5 | Render briefing PDF | pending |
| B6 | Phase-04 prelude artifact | pending |
| B7 | Commit | pending |

## What landed in B3b

`~/code/herald/_project/ov-1/core/upload.typ` — final form (5.3 KB). Renders to `core/upload-{light,dark}.svg` (149 KB each). Two iterations against the user before lock:

- *Iteration 1* — initial render from the planned skeleton. User asked for two adjustments.
- *Iteration 2* — applied user adjustments and locked.

### Adjustments applied during B3b

1. **Return-edge bend direction.** Initial `bend: -25deg` curved the dashed return *upward* relative to the forward edge — visually fought the baseline flow. Flipped to `bend: 25deg` so the return curves *downward* below the forward edges. Note: in Fletcher 0.5.x with default y-axis orientation, **positive `bend` curves toward larger y (downward on a standard top-origin canvas)** — invert any future return-edge bend that initially reads "wrong way".
2. **Upload-client description.** Final wording (per user direction):
   > Submits documents and their metadata (V2 ID, deployment instance) to Herald for classification.

   The V2 ID and deployment instance are upstream identifiers Herald accepts on upload — surfacing them in the actor description tells the leadership audience what the client is actually handing over, without leaking implementation chrome.

### Final layout

```
   <client>(-2, 0) ──upload document──▶ <h>(0, 0) ──store immutably──▶ <blob>(2, -0.5)
                                            │
                                            └──register record────▶ <pg>(2, 0.5)
                ◀── return identifier (dashed, bend 25deg, beneath) ──
```

### Final edge inventory

| Edge | Mark | Label | Stroke |
|---|---|---|---|
| `<client.east> → <h.west>` | `->` | `upload document` | solid green |
| `<h.east> → <blob.west>` | `->` | `store immutably` | solid green |
| `<h.east> → <pg.west>` | `->` | `register record` | solid green |
| `<h.south-west> → <client.south-east>` | `-->` | `return identifier` | dashed green, `bend: 25deg` |

## Conventions confirmed or surfaced in B3b

These are *additions and refinements* on top of the B3a conventions in `checkpoint-1.md`. Read both checkpoints when resuming.

### 1. Step-and-actor flow with retained edge labels

Confirmed adaptation from `checkpoint-1.md` §*Adaptation notes*: when relationships are step *names* (handoffs / transitions) rather than dependency *contexts*, edge labels carry the semantics and shape descriptions stay lighter than the dense readme.typ blocks. In B3b: shape descriptions are 1–4 short lines (actor identity / role); edge labels are 2-word imperative verb phrases (`upload document`, `store immutably`, `register record`, `return identifier`).

This stays true for B3c–B3e (state-graph card and milestone strips). For state graphs, transition labels are even more load-bearing (the conditional fork in `classification` is the *concept*).

### 2. Native Fletcher `label-fill` is canonical

`checkpoint-1.md` §6 documented a custom `lbl()` `box(inset: ...)` helper as a fallback. B3b adopts the native form per the labels-and-encapsulation reference:

```typst
edge(
  <a>, <b>, "->",
  step-label("..."),
  label-pos: 0.5,
  label-side: center,
  label-sep: tokens.label-sep,
  label-fill: palette.surface,
  stroke: edge-stroke,
)
```

`step-label(s)` wraps `text(...)` with `size: tokens.size-label`, `weight: tokens.weight-light`, `fill: palette.ink-muted`, `style: "italic"` for stakeholder-voiced step labels. Reuse this helper in B3c–B3e.

The custom `lbl()` `box` from checkpoint-1 §6 is still valid for situations where the native `label-fill` doesn't give enough breathing room — keep it as a fallback, not the default.

### 3. Vertical fan-out from a focal node

When a focal node has parallel writes / reads to multiple stores, stack the destinations vertically at the same x and let edges fan out from a shared anchor (`<h.east>` here, both edges):

```typst
kinded((2, -0.5), ..., ref: <blob>),  // top destination
kinded((2,  0.5), ..., ref: <pg>),    // bottom destination

edge(<h.east>, <blob.west>, "->", step-label("store immutably"), ...),
edge(<h.east>, <pg.west>,   "->", step-label("register record"), ...),
```

Fletcher routes from the same anchor without crossing because the destination y-positions force the divergence. The labels at midpoint (`label-pos: 0.5`) sit at distinct y-values and don't collide. This is reusable for any "one-to-many" handoff in flow diagrams.

### 4. Dashed + bent-below return edge for response semantics

Pattern for response / acknowledgement / return-value edges that shouldn't compete with the forward flow:

```typst
edge(<source.south-west>, <dest.south-east>, "-->",
  step-label("..."),
  bend: 25deg,                    // positive = curves toward larger y
  label-pos: 0.5, label-side: center,
  label-sep: tokens.label-sep,
  label-fill: palette.surface,
  stroke: (
    paint: palette.green.stroke,
    thickness: tokens.stroke-default,
    dash: "dashed",
  ),
)
```

Anchoring at `south-*` corners + positive bend produces a curve that drops below the forward baseline. The dashed body + the offset-from-baseline together encode the return semantics at a glance.

**Sign convention:** positive `bend` curves toward larger y-axis values. With Fletcher's default top-origin canvas, that means *downward* on screen. Don't trust intuition — flip if the first render reads wrong, as happened in B3b iteration 2.

### 5. Helper copy-paste over shared module

`kinded` is now duplicated across `readme.typ` and `upload.typ` (verbatim). Defer extraction to a shared `_helpers.typ` until after B3e. Rationale: the four flow diagrams may surface variants (state-with-transition-fork-handle, milestone-with-trigger-event) the readme.typ helper doesn't accommodate; extracting too early forces premature signature decisions. Re-evaluate at B3e once the visual vocabulary is fully exercised.

### 6. Cross-diagram visual cohesion

Per findings open-question Q3 (resolved 2026-04-29): the briefing's diagrams share a visual vocabulary so they read as one set:

- **Hue assignments stay consistent.** Document/upload client → orange (upstream actor). Herald → blue (focal). Azure managed services → coral. Reviewer → orange (also human actor). Future flow diagrams should reuse these hue assignments for the same conceptual entity classes.
- **Glyphs stay consistent.** Same Font Awesome codepoints for the same entity class: `\u{F007}` for actors, `\u{F0AC}` for Herald, `\u{F0A0}` for blob storage, `\u{F1C0}` for databases, `\u{F0C2}` for AI services, `\u{F1B2}` for runtime/deployment.
- **Edge stroke baseline.** `tokens.stroke-default + palette.green.stroke` for forward flow. Variants (dashed, double, mark-style) signal *kind* of relationship, not different palettes.

This is the "single design system, per-diagram visual choices" principle from the brief — applied concretely.

## Adaptation notes for B3c–B3e

Refined from checkpoint-1.md based on what B3b confirmed:

| File | Visual approach (per findings) | Convention adaptation |
|---|---|---|
| `core/readme.typ` | hub-and-spoke (compose fresh) — **done** | shape-body-as-semantic + container split |
| `core/upload.typ` | step-and-actor flow — **done** | `kinded` shapes + retained edge labels + dashed/bent return |
| `core/classification.typ` | state-graph card with fork-and-rejoin | `kinded` shapes for states (lighter than upload's actor descriptions); transition labels CRITICAL — the conditional fork is the *concept*. May need a third edge style (e.g., dotted or different mark) to differentiate the conditional branches from the unconditional transitions. Consider `?`-prefixed labels (`if any page flagged`, `if no page flagged`) at the fork. |
| `core/release.typ` | labelled milestone strip | Horizontal pipeline; reuse `kinded` for milestone stops. Edge labels carry trigger names (`push tag`, `build & push`, `publish`). External terminal nodes (GHCR, public release page) are coral. The published image is the focal terminal — heavier hue or stroke. |
| `core/cds-release.typ` | labelled milestone strip with focal step | Same as `release` but with the cross-domain transfer rendered as the focal step (heavier stroke / hue accent). Two-band background (public-side / secure-side) is the natural place to reuse the two-node container pattern from B3a. Sensitivity rule applies: external services rendered as *class* names only (`GitHub Container Registry`, `cross-domain transfer service`, `secure storage destination`), no tenant/account/runner names. |

The shape-body-as-semantic convention from B3a remains valid for *all* diagrams — actors and entities still carry identity in body. What varies per diagram is whether **edge labels** also carry weight: `readme.typ` says no (dependency context lives in shape body), `upload.typ` says yes (handoffs are the concept), `classification.typ` says yes-and-load-bearing (the fork is the focus), `release` and `cds-release` say yes (trigger names matter).

## Resume protocol

When resuming in a fresh context:

1. Read this checkpoint (`checkpoint-2.md`).
2. Read `~/code/herald/.claude/context/sessions/2026-04-29-herald-ov1/checkpoint-1.md` for B3a conventions (shape-body-as-semantic, two-node container pattern, cardinal anchors, custom `lbl()` fallback).
3. Read `~/code/herald/.claude/plans/initialize-tau-herald-overview-brief-md-reactive-crane.md` — the original session plan.
4. Read `~/tau/herald-overview-brief.md` — revised brief; locked decisions and the B3a addendum live here.
5. Read `~/code/herald/.claude/context/sessions/2026-04-29-herald-ov1/findings.md` — technical-writer per-diagram findings (resume context for B3c–B3e).
6. Skim `~/code/herald/_project/ov-1/core/readme.typ` and `~/code/herald/_project/ov-1/core/upload.typ` — the `kinded` helper and edge-label conventions for reuse.
7. Confirm with the user before resuming. Default next step: B3c (`core/classification.typ`) under a new plan-mode-driven design pass.

## Pending toolkit updates

These accumulate across the session and apply post-B7 to `~/tau/diagrams/`:

(*from checkpoint-1*)
1. `~/tau/diagrams/.claude/skills/diagram-ingredients/references/labels-and-encapsulation.md` — Add the two-node container variant for header-above-inner-nodes.
2. `~/tau/diagrams/.claude/skills/diagram-authoring/SKILL.md` or new reference — The "shape-body-as-semantic" pattern: when to choose body-borne semantics vs edge labels in dense diagrams.
3. `~/tau/herald-overview-brief.md` — Brief addendum: conventions established during B3a, for the four remaining diagrams.

(*new in B3b*)
4. `~/tau/diagrams/.claude/skills/diagram-ingredients/references/edges-and-marks.md` — Add the **dashed + bent-below return-edge pattern** (with the bend-sign convention note: positive bend curves toward larger y).
5. `~/tau/diagrams/.claude/skills/diagram-ingredients/references/edges-and-marks.md` — Add the **vertical fan-out from a focal node** pattern (one-to-many parallel writes; stacked destinations + shared anchor).
6. `~/tau/diagrams/.claude/skills/diagram-ingredients/references/labels-and-encapsulation.md` — Promote the **native Fletcher `label-fill`** form as the default; demote the custom `lbl()` box helper to fallback.
7. `~/tau/diagrams/.claude/skills/typst-diagrams/references/fletcher-pitfalls.md` — Document the **`bend` sign convention** (positive = curve toward larger y on default-orientation canvas).

Defer all toolkit edits until after B7. The phase-04 prelude artifact (B6) carries the conventions forward; toolkit edits are a separate session.

## Critical files

| Path | Role |
|---|---|
| `~/code/herald/_project/ov-1/core/readme.typ` | B3a artifact; source of `kinded` helper |
| `~/code/herald/_project/ov-1/core/readme-{light,dark}.svg` | Rendered B3a |
| `~/code/herald/_project/ov-1/core/upload.typ` | **B3b artifact** |
| `~/code/herald/_project/ov-1/core/upload-{light,dark}.svg` | **Rendered B3b** |
| `~/code/herald/_project/ov-1/{mise.toml,design/}` | Render pipeline + design layer mirrored from TAU |
| `~/code/herald/.claude/agents/technical-writer.md` | Subagent symlink → `~/tau/diagrams/.claude/agents/technical-writer.md` |
| `~/code/herald/.claude/context/sessions/2026-04-29-herald-ov1/findings.md` | Per-diagram structured findings |
| `~/code/herald/.claude/context/sessions/2026-04-29-herald-ov1/checkpoint-1.md` | B3a conventions (still authoritative) |
| `~/code/herald/.claude/context/sessions/2026-04-29-herald-ov1/checkpoint-2.md` | This file |
| `~/code/herald/.claude/plans/initialize-tau-herald-overview-brief-md-reactive-crane.md` | Original session plan |
| `~/code/herald/.claude/plans/given-how-heavy-the-greedy-dongarra.md` | B3b plan |
| `~/tau/herald-overview-brief.md` | Revised brief + B3a addendum |
| `~/tau/diagrams/.claude/project/04-advanced-tau-diagrams.md` | Phase-04 plan |
| `~/tau/diagrams/.claude/agents/technical-writer.md` | Canonical subagent contract |
| `~/tau/diagrams/.claude/skills/tau-diagrams/SKILL.md` | TAU diagram standard |
| `~/tau/diagrams/.claude/skills/diagram-ingredients/references/labels-and-encapsulation.md` | Container pattern reference |
| `~/tau/diagrams/.claude/skills/diagram-ingredients/references/edges-and-marks.md` | Edges + marks (waypoints fixed earlier this session) |
| `~/tau/diagrams/.claude/skills/typst-diagrams/references/fletcher-pitfalls.md` | Fletcher pitfalls (route-string parsing fixed earlier this session) |

## Open thread

(*carried from checkpoint-1*)

1. **Mark overlay on shape.** Tried literal coord destination + `layer: 1`; both produced clipped arrows. Workaround was to switch the mark style (`-O`) so the visual sits naturally at the boundary. If a future diagram needs an arrow tip *inside* a shape, this remains unsolved.
2. **Header-vs-shape rendering layer in containers.** The two-node container pattern works because header and inner shapes don't spatially overlap. If a future container needs a header that overlaps the inner shapes vertically (e.g., compact dense container), this will need re-investigation.

(*new in B3b*)

3. **Bend-sign default intuition mismatch.** Both initial bend choices in B3b's design phase (the plan and the first implementation) used negative bend assuming "below = negative". Fletcher's actual convention (positive = larger y on default canvas) caught the iteration cycle. Toolkit-update item #7 captures this; until then, expect to flip-test on first render for any bent edge.

Note all three for the phase-04 prelude artifact (B6).
