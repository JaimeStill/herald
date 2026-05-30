// classification — how Herald analyses a document, for the slide that primes
// the live demo.
//
// Terminal-native style (see design/README.md): outline-only shapes, one accent
// per entity kind. Accents LOCKED:
//   workflow states = violet (Herald's interior) · Azure services = blue.
//
// Focal concept: the conditional fork-and-rejoin at Classify. Some documents
// take a re-render-and-re-classify detour through Enhance; others go straight
// to Finalize. Visual encoding:
//   - solid edges  = unconditional transitions and service calls
//   - dashed edges = the two conditional fork branches
//
// Layout:
//   [Blob]                        [AI Foundry]
//      │                               │
//      ▼               ┌───────────────┼───────────────┐
//   ┌──────────────────┴────────────────────────────┐  │
//   │ [Init]──▶[Classify]┄┄(if any flagged)┄▶[Enhance]│ │
//   │                 ┊                          │     │
//   │                 ┊┄┄(if none flagged)┄┄┄┄┄┄▶│     │
//   │                                        [Finalize]│
//   └──────────────────────────────────────────┬───────┘
//                                               ▼
//                                          [PostgreSQL]

#import "@preview/fletcher:0.5.8" as fletcher: diagram, node, edge
#import "design/tokens.typ": tokens
#import "design/theme.typ": palette

#set page(width: auto, height: auto, margin: tokens.pad-inside-container, fill: palette.surface)
#set text(font: tokens.font, fill: palette.ink)

// This diagram is wide and slots into a single slide, so it is scaled down hard
// on the page. Bump the type up locally (only here, not in shared tokens.typ)
// so labels stay legible after that downscale.
#let scale = 1.7
#let sz-glyph = 18pt * scale
#let sz-title = tokens.size-body * scale
#let sz-label = tokens.size-label * scale

// ---- Shape helper (flat): accent stroke + accent title, no fill -----------

#let kinded(pos, accent, glyph, title, kind, description, ref: none, w: auto, fill: none) = node(pos,
  grid(columns: (auto, auto), column-gutter: tokens.gap-cell,
    align: (left + top, left + top),
    text(size: sz-glyph, fill: accent, glyph),
    block(width: w, stack(dir: ttb, spacing: tokens.gap-structured-text,
      block(width: 100%, align(center,
        text(size: sz-title, weight: tokens.weight-bold, fill: accent, title))),
      block(width: 100%, align(center,
        text(size: sz-label, weight: tokens.weight-light, fill: palette.ink-muted,
          style: "italic", "(" + kind + ")"))),
      block(spacing: 0pt, line(length: 100%, stroke: tokens.stroke-thin + palette.border)),
      text(size: sz-label, fill: palette.ink, description),
    )),
  ),
  shape: fletcher.shapes.rect,
  fill: fill,
  stroke: tokens.stroke-default + accent,
  inset: tokens.pad-inside-shape,
  corner-radius: tokens.radius-shape,
  name: ref,
)

// ---- Edge styling --------------------------------------------------------

#let edge-stroke = tokens.stroke-default + palette.ink-subtle
#let edge-stroke-dashed = (
  paint: palette.ink-subtle,
  thickness: tokens.stroke-default,
  dash: "dashed",
)

#let step-label(s) = text(
  size: sz-label, weight: tokens.weight-light,
  fill: palette.ink-muted, style: "italic", s,
)

// ---- Diagram -------------------------------------------------------------

#diagram(
  spacing: (1.7 * tokens.space-between-shapes, 1.35 * tokens.space-between-ranks),

  // ---- Workflow states (violet = Herald's interior) -----------------------

  // Glyph: \u{F019} = download — fetch + render.
  kinded((-3, 0), palette.violet, "\u{F019}",
    "Init", "workflow state",
    [Fetches the document \
     from durable storage \
     and renders each page \
     as an image.],
    w: 240pt, ref: <init>),

  // Glyph: \u{F002} = magnifier — page-by-page inspection.
  kinded((-1, 0), palette.violet, "\u{F002}",
    "Classify", "workflow state",
    [Sends each page image \
     to the AI service and \
     records the markings \
     it sees.],
    w: 240pt, ref: <classify>),

  // Glyph: \u{F042} = adjust — brightness / contrast / saturation tuning.
  kinded((1, -1), palette.violet, "\u{F042}",
    "Enhance", "conditional state",
    [Re-renders flagged pages \
     with image adjustments \
     and re-classifies them \
     when a first look was \
     inconclusive.],
    w: 240pt, ref: <enhance>),

  // Glyph: \u{F0AE} = tasks — synthesis / closure.
  kinded((3, 0), palette.violet, "\u{F0AE}",
    "Finalize", "workflow state",
    [Synthesises per-page \
     results into a document- \
     level classification, \
     confidence, and rationale.],
    w: 240pt, ref: <finalize>),

  // ---- External services (blue = Azure managed) --------------------------

  // Glyph: \u{F0A0} = hard-drive — object storage.
  kinded((-3, -2), palette.blue, "\u{F0A0}",
    "Azure Blob Storage", "object store",
    [Source of the \
     document bytes.],
    w: 240pt, ref: <blob>),

  // Glyph: \u{F0C2} = cloud — AI service.
  kinded((1, -2.5), palette.blue, "\u{F0C2}",
    "Azure AI Foundry", "AI service",
    [Per-page and document- \
     level vision inference.],
    w: 240pt, ref: <ai>),

  // Glyph: \u{F1C0} = database.
  kinded((3, 1.5), palette.blue, "\u{F1C0}",
    "Azure PostgreSQL", "database",
    [Records the final \
     classification.],
    w: 240pt, ref: <pg>),

  // ---- Workflow band (surface-muted enclosure) ---------------------------

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

  // Init → Classify: unconditional, solid, no label.
  edge(<init.east>, <classify.west>, "->", stroke: edge-stroke),

  // Classify → Enhance: conditional, dashed.
  edge(<classify.north-east>, <enhance.south-west>, "-->",
    step-label("if any page flagged"),
    bend: -20deg,
    label-pos: 0.5, label-side: center,
    label-sep: tokens.label-sep,
    label-fill: palette.surface,
    stroke: edge-stroke-dashed),

  // Classify → Finalize: conditional, dashed, straight along baseline.
  edge(<classify.east>, <finalize.west>, "-->",
    step-label("if none flagged"),
    label-pos: 0.5, label-side: center,
    label-sep: tokens.label-sep,
    label-fill: palette.surface,
    stroke: edge-stroke-dashed),

  // Enhance → Finalize: unconditional rejoin, solid, no label.
  edge(<enhance.south-east>, <finalize.north-west>, "->", stroke: edge-stroke),

  // ---- Service calls -----------------------------------------------------

  // Blob Storage → Init: fetch document bytes.
  edge(<blob.south>, <init.north>, "->",
    step-label("fetch document"),
    label-pos: 0.5, label-side: center,
    label-sep: tokens.label-sep,
    label-fill: palette.surface,
    stroke: edge-stroke),

  // AI Foundry → Classify: per-page vision (parallel).
  edge(<ai.south-west>, <classify.north>, "->",
    step-label("vision per page"),
    label-pos: 0.5, label-side: center,
    label-sep: tokens.label-sep,
    label-fill: palette.surface,
    stroke: edge-stroke),

  // AI Foundry → Enhance: per-flagged-page re-vision.
  edge(<ai.south>, <enhance.north>, "->",
    step-label("vision per flagged page"),
    label-pos: 0.5, label-side: center,
    label-sep: tokens.label-sep,
    label-fill: palette.surface,
    stroke: edge-stroke),

  // AI Foundry → Finalize: document-level synthesis (one chat call).
  edge(<ai.south-east>, <finalize.north>, "->",
    step-label("synthesize classification"),
    label-pos: 0.5, label-side: center,
    label-sep: tokens.label-sep,
    label-fill: palette.surface,
    stroke: edge-stroke),

  // Finalize → PostgreSQL: persistence.
  edge(<finalize.south>, <pg.north>, "->",
    step-label("persist classification"),
    label-pos: 0.5, label-side: center,
    label-sep: tokens.label-sep,
    label-fill: palette.surface,
    stroke: edge-stroke),
)
