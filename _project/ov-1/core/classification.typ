// core/classification — How Herald analyses a document (stakeholder voice).
//
// State-graph card: four workflow states (Init, Classify, Enhance, Finalize)
// inside a surface-muted band that anchors them as Herald's interior.
// External services (Blob Storage, AI Foundry, PostgreSQL) hover outside
// the band with edges into the calling states.
//
// Focal concept: the conditional fork-and-rejoin at Classify. Some documents
// take a re-render-and-re-classify detour through Enhance; others go
// straight to Finalize. Visual encoding:
//   - solid edges = unconditional transitions and service calls
//   - dashed edges = the two conditional fork branches
//
// Layout:
//
//   [Blob]                       [AI Foundry]
//      │                              │
//      ▼              ┌───────────────┼───────────────┐
//   ┌──────────────┐  │               ▼               │
//   │ [Init]──▶[Classify]┄┄(if any flagged)┄▶[Enhance]│
//   │                ┊                          │     │
//   │                ┊┄┄(if no flagged)┄┄┄┄┄┄┄┄▶│     │
//   │                                       [Finalize]│
//   └─────────────────────────────────────────┬───────┘
//                                              ▼
//                                          [PostgreSQL]
//
// `kinded` / `step-label` / edge-stroke helpers are copied verbatim from
// upload.typ; will be extracted to a shared module after B3e per
// checkpoint-2.md §5.

#import "@preview/fletcher:0.5.8" as fletcher: diagram, node, edge
#import "../design/tokens.typ": tokens
#import "../design/theme.typ": palette

#set page(width: auto, height: auto, margin: tokens.pad-inside-container, fill: palette.surface)
#set text(font: tokens.font, fill: palette.ink)

// ---- Shape helper --------------------------------------------------------

// `kinded` — body shape with glyph + stacked(title / (kind) / divider /
// description). Title and (kind) centred within the body's natural width;
// description stays left-aligned. Optional `extras` content appended below.
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

// ---- Edge styling --------------------------------------------------------

#let edge-stroke = tokens.stroke-default + palette.green.stroke
#let edge-stroke-dashed = (
  paint: palette.green.stroke,
  thickness: tokens.stroke-default,
  dash: "dashed",
)

#let step-label(s) = text(
  size: tokens.size-label,
  weight: tokens.weight-light,
  fill: palette.ink-muted,
  style: "italic",
  s,
)

// ---- Diagram -------------------------------------------------------------

#diagram(
  spacing: (3 * tokens.space-between-shapes, 2.5 * tokens.space-between-shapes),

  // ---- Workflow states (purple = Herald's interior) -----------------------

  // Glyph: \u{F019} = download — fetch + render.
  kinded((-3, 0), palette.purple, "\u{F019}",
    "Init", "workflow state",
    [Fetches the document \
     from durable storage \
     and renders each page \
     as an image.],
    ref: <init>,
  ),

  // Glyph: \u{F002} = magnifier — page-by-page inspection.
  kinded((-1, 0), palette.purple, "\u{F002}",
    "Classify", "workflow state",
    [Sends each page image \
     to the AI service and \
     records the markings \
     it sees.],
    ref: <classify>,
  ),

  // Glyph: \u{F042} = adjust — brightness / contrast / saturation tuning.
  kinded((1, -1), palette.purple, "\u{F042}",
    "Enhance", "conditional state",
    [Re-renders flagged pages \
     with image adjustments \
     and re-classifies them \
     when a first look was \
     inconclusive.],
    ref: <enhance>,
  ),

  // Glyph: \u{F0AE} = tasks — synthesis / closure.
  kinded((3, 0), palette.purple, "\u{F0AE}",
    "Finalize", "workflow state",
    [Synthesises per-page \
     results into a document- \
     level classification, \
     confidence, and rationale.],
    ref: <finalize>,
  ),

  // ---- External services (coral = Azure managed) -------------------------

  // Glyph: \u{F0A0} = hard-drive — object storage. Matches readme.typ + upload.typ.
  kinded((-3, -2), palette.coral, "\u{F0A0}",
    "Azure Blob Storage", "object store",
    [Source of the \
     document bytes.],
    ref: <blob>,
  ),

  // Glyph: \u{F0C2} = cloud — AI service. Matches readme.typ.
  kinded((1, -2.5), palette.coral, "\u{F0C2}",
    "Azure AI Foundry", "AI service",
    [Per-page and document- \
     level vision and classification synthesis.],
    ref: <ai>,
  ),

  // Glyph: \u{F1C0} = database. Matches readme.typ + upload.typ.
  kinded((3, 1.5), palette.coral, "\u{F1C0}",
    "Azure PostgreSQL", "database",
    [Records the final \
     classification.],
    ref: <pg>,
  ),

  // ---- Workflow band (surface-muted enclosure) ---------------------------

  // Single-node container — no header, just fill. Anchors the four states
  // visually as "Herald's interior" without competing with the conditional
  // fork for focal weight. Two-node split (B3a) is unnecessary here.
  node(
    enclose: (<init>, <classify>, <enhance>, <finalize>),
    [],
    shape: fletcher.shapes.rect,
    fill: palette.surface-muted,
    stroke: none,
    inset: tokens.pad-inside-container,
    corner-radius: tokens.radius-container,
    snap: -1,
  ),

  // ---- State transitions -------------------------------------------------

  // Init → Classify: unconditional, solid green, no label.
  edge(<init.east>, <classify.west>, "->", stroke: edge-stroke),

  // Classify → Enhance: conditional, dashed, bend toward Enhance's offset.
  // Negative bend curves toward smaller y (upward on default canvas) per
  // checkpoint-2 §3 sign convention; flip on render if it reads wrong.
  edge(<classify.north-east>, <enhance.south-west>, "-->",
    step-label("if any page flagged"),
    bend: -20deg,
    label-pos: 0.5, label-side: center,
    label-sep: tokens.label-sep,
    label-fill: palette.surface,
    stroke: edge-stroke-dashed,
  ),

  // Classify → Finalize: conditional, dashed, straight along baseline.
  edge(<classify.east>, <finalize.west>, "-->",
    step-label("if no page flagged"),
    label-pos: 0.5, label-side: center,
    label-sep: tokens.label-sep,
    label-fill: palette.surface,
    stroke: edge-stroke-dashed,
  ),

  // Enhance → Finalize: unconditional rejoin, solid green, no label.
  edge(<enhance.south-east>, <finalize.north-west>, "->", stroke: edge-stroke),

  // ---- Service calls -----------------------------------------------------

  // Blob Storage → Init: fetch document bytes (only Init touches Blob).
  edge(<blob.south>, <init.north>, "->",
    step-label("fetch document"),
    label-pos: 0.5, label-side: center,
    label-sep: tokens.label-sep,
    label-fill: palette.surface,
    stroke: edge-stroke,
  ),

  // AI Foundry → Classify: per-page vision (parallel).
  edge(<ai.south-west>, <classify.north>, "->",
    step-label("vision per page"),
    label-pos: 0.5, label-side: center,
    label-sep: tokens.label-sep,
    label-fill: palette.surface,
    stroke: edge-stroke,
  ),

  // AI Foundry → Enhance: per-flagged-page re-vision.
  edge(<ai.south>, <enhance.north>, "->",
    step-label("vision per flagged page"),
    label-pos: 0.5, label-side: center,
    label-sep: tokens.label-sep,
    label-fill: palette.surface,
    stroke: edge-stroke,
  ),

  // AI Foundry → Finalize: document-level synthesis (one chat call).
  edge(<ai.south-east>, <finalize.north>, "->",
    step-label("synthesize classification"),
    label-pos: 0.5, label-side: center,
    label-sep: tokens.label-sep,
    label-fill: palette.surface,
    stroke: edge-stroke,
  ),

  // Finalize → PostgreSQL: persistence (post-workflow, abstracted to
  // Finalize's outbound for stakeholder voice).
  edge(<finalize.south>, <pg.north>, "->",
    step-label("persist classification"),
    label-pos: 0.5, label-side: center,
    label-sep: tokens.label-sep,
    label-fill: palette.surface,
    stroke: edge-stroke,
  ),
)
