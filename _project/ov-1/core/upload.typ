// core/upload — How a document enters Herald (stakeholder voice).
//
// Step-and-actor flow: actors on a baseline, edges carry handoff labels.
// The actor handoff IS the concept. Three forward edges fan out from
// Herald — one inbound from the upload client, two outbound to the
// stores. A dashed return edge runs beneath the forward flow to carry
// the registered identifier back to the client.
//
// Layout:
//                                   ┌─▶ [Azure Blob Storage]
//   [Upload client] ─▶ [Herald] ────┤
//                ◀─ ─ ─ ─ ─ ─ ─ ─ ─ └─▶ [Azure PostgreSQL]
//
// `kinded` is copied verbatim from core/readme.typ. After B3e completes
// the helper may move to a shared module; for now each diagram stays
// self-contained until the signature is validated across all five.

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

  // Glyph: \u{F007} = user. Reused from readme.typ for cross-diagram
  // cohesion with the document-source actor.
  kinded((-2, 0), palette.orange, "\u{F007}",
    "Upload client", "upstream",
    [Submits documents and their \
     metadata (V2 ID, deployment \
     instance) to Herald for \
     classification.],
    ref: <client>,
  ),

  // Glyph: \u{F0AC} = globe — service identity. Matches readme.typ.
  kinded((0, 0), palette.blue, "\u{F0AC}",
    "Herald", "service",
    [Receives uploads; durably stores \
     each document and registers it \
     for classification.],
    ref: <h>,
  ),

  // Glyph: \u{F0A0} = hard-drive — object storage. Matches readme.typ.
  kinded((2, -0.5), palette.coral, "\u{F0A0}",
    "Azure Blob Storage", "object storage",
    [Durable, immutable document store.],
    ref: <blob>,
  ),

  // Glyph: \u{F1C0} = database. Matches readme.typ.
  kinded((2, 0.5), palette.coral, "\u{F1C0}",
    "Azure PostgreSQL", "database",
    [Document registration record.],
    ref: <pg>,
  ),

  // ---- Forward flow ------------------------------------------------------

  // Client → Herald: the inbound handoff.
  edge(<client.east>, <h.west>, "->",
    step-label("upload document"),
    label-pos: 0.5, label-side: center,
    label-sep: tokens.label-sep,
    label-fill: palette.surface,
    stroke: edge-stroke,
  ),

  // Herald → Blob Storage: durable bytes.
  edge(<h.east>, <blob.west>, "->",
    step-label("store immutably"),
    label-pos: 0.5, label-side: center,
    label-sep: tokens.label-sep,
    label-fill: palette.surface,
    stroke: edge-stroke,
  ),

  // Herald → PostgreSQL: registration record.
  edge(<h.east>, <pg.west>, "->",
    step-label("register record"),
    label-pos: 0.5, label-side: center,
    label-sep: tokens.label-sep,
    label-fill: palette.surface,
    stroke: edge-stroke,
  ),

  // ---- Return flow -------------------------------------------------------

  // Herald → Client (response): dashed + bent below the forward edge so
  // the return semantics reads at a glance without competing for the
  // baseline visual weight.
  edge(<h.south-west>, <client.south-east>, "-->",
    step-label("return identifier"),
    bend: 25deg,
    label-pos: 0.5, label-side: center,
    label-sep: tokens.label-sep,
    label-fill: palette.surface,
    stroke: edge-stroke-dashed,
  ),
)
