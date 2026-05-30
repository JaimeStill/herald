// overview — Herald architecture overview for the Overview slide.
//
// Terminal-native style (see design/README.md): outline-only shapes on the
// terminal background, one accent per entity kind used as stroke + title, no
// chromatic fills. Entity accents are LOCKED:
//   Herald = violet (focal) · actors = cyan · Azure managed services = blue.
//
// Layout:
//   [Document source]                                   [Reviewer]
//                             [Herald]
//   ┌─── Azure Managed Services ─────────────────────────────────┐
//   │ [AI Foundry] [PostgreSQL] [Blob Storage]   [Container Apps] │
//   └────────────────────────────────────────────────────────────┘
//
// Each shape carries icon-left + stacked(title / (kind) / divider /
// description); the description articulates the entity's role relative to
// Herald, so edges stay connectors only (direction matters, not labels).

#import "@preview/fletcher:0.5.8" as fletcher: diagram, node, edge
#import "design/tokens.typ": tokens
#import "design/theme.typ": palette

#set page(width: auto, height: auto, margin: tokens.pad-inside-container, fill: palette.surface)
#set text(font: tokens.font, fill: palette.ink)

// ---- Shape helper (flat): accent stroke + accent title, no fill -----------

// `w` forces a uniform body width (used to even the Azure service row and keep
// titles on one line); `fill` lets a node occlude an edge mark it docks onto.
#let kinded(pos, accent, glyph, title, kind, description, ref: none, w: auto, fill: none) = node(pos,
  grid(columns: (auto, auto), column-gutter: tokens.gap-cell,
    align: (left + top, left + top),
    text(size: 18pt, fill: accent, glyph),
    block(width: w, stack(dir: ttb, spacing: tokens.gap-structured-text,
      block(width: 100%, align(center,
        text(size: tokens.size-body, weight: tokens.weight-bold, fill: accent, title))),
      block(width: 100%, align(center,
        text(size: tokens.size-label, weight: tokens.weight-light, fill: palette.ink-muted,
          style: "italic", "(" + kind + ")"))),
      block(spacing: 0pt, line(length: 100%, stroke: tokens.stroke-thin + palette.border)),
      text(size: tokens.size-label, fill: palette.ink, description),
    )),
  ),
  shape: fletcher.shapes.rect,
  fill: fill,
  stroke: tokens.stroke-default + accent,
  inset: tokens.pad-inside-shape,
  corner-radius: tokens.radius-shape,
  name: ref,
)

#let edge-stroke = tokens.stroke-default + palette.ink-subtle

// ---- Diagram --------------------------------------------------------------

#diagram(
  spacing: (3 * tokens.space-between-shapes, 2.5 * tokens.space-between-shapes),

  // ---- Top row: actors ----------------------------------------------------

  kinded((-0.8, 0.5), palette.cyan, "\u{F007}",
    "Document source", "upstream",
    [Submits documents \
     for classification.],
    ref: <doc>),

  kinded((0.8, 0.5), palette.cyan, "\u{F007}",
    "Reviewer", "human",
    [Validates low-confidence \
     classifications.],
    ref: <rev>),

  // ---- Centre: Herald (focal) --------------------------------------------

  kinded((0, 1), palette.violet, "\u{F0AC}",
    "Herald", "service",
    [Classifies security markings \
     on DoD documents. \
      \
     Users sign in via \
     Microsoft Entra ID. \
      \
     Single Go binary; web client \
     embedded and served at /app.],
    ref: <h>),

  // ---- Bottom: Azure Managed Services container --------------------------

  // Uniform body width (`w`) evens the row and keeps every title on one line.
  // Glyph: \u{F0C2} = cloud — AI service compute.
  kinded((-1, 3), palette.blue, "\u{F0C2}",
    "Azure AI Foundry", "AI service",
    [Per-page and document-level \
     vision inference.],
    w: 150pt, ref: <ai>),
  // Glyph: \u{F1C0} = database.
  kinded((-0.33, 3), palette.blue, "\u{F1C0}",
    "Azure PostgreSQL", "database",
    [Document metadata and \
     classification records.],
    w: 150pt, ref: <pg>),
  // Glyph: \u{F0A0} = hard-drive — object storage.
  kinded((0.33, 3), palette.blue, "\u{F0A0}",
    "Azure Blob Storage", "object storage",
    [Durable document \
     blob storage.],
    w: 150pt, ref: <blob>),
  // Glyph: \u{F1B2} = cube — runtime/deployment.
  kinded((1, 3), palette.blue, "\u{F1B2}",
    "Azure Container Apps", "runtime",
    [Herald's deployment host.],
    w: 150pt, ref: <aca>),

  // Container header — own node above the inner shapes so it paints in the
  // foreground over the container's `snap: -1` fill.
  node((0, 2.3),
    stack(dir: ttb, spacing: tokens.gap-structured-text,
      grid(columns: (auto, auto), column-gutter: tokens.gap-cell,
        align: (left + horizon, left + horizon),
        text(size: 18pt, fill: palette.ink-muted, "\u{F0C2}"),
        text(size: tokens.size-body, weight: tokens.weight-bold, fill: palette.ink,
          "Azure Managed Services"),
      ),
      text(size: tokens.size-label, fill: palette.ink,
        "User-managed identity authorizes Herald's service connections."),
    ),
    shape: fletcher.shapes.rect,
    fill: none, stroke: none, inset: 0pt,
    name: <azure-header>),

  // Container background — encloses header + four inner shapes; fill only,
  // no border, behind everything via snap: -1.
  node(
    enclose: (<azure-header>, <ai>, <pg>, <blob>, <aca>),
    name: <azure>,
    [],
    shape: fletcher.shapes.rect,
    fill: palette.surface-muted,
    stroke: none,
    inset: tokens.pad-inside-container,
    corner-radius: tokens.radius-container,
    snap: -1),

  // ---- Edges (connectors only) -------------------------------------------

  // Document source → Herald (drop, then right into Herald's west).
  edge(<doc.south>, (-0.8, 1), <h.west>, "->", stroke: edge-stroke),
  // Reviewer ↔ Herald (drop, then left into Herald's east).
  edge(<rev.south>, (0.8, 1), <h.east>, "<->", stroke: edge-stroke),
  // Herald → Azure Managed Services (bundled "uses these services").
  edge(<h.south>, <azure>, "->", stroke: edge-stroke),
  // Herald → Container Apps (specific deployment target). Solid stealth head
  // stays distinct from the open `->` data-flow barbs while reading cleanly as
  // "deploys to". Default layer so it draws above the container background.
  edge(<h.south-east>, <aca.north>, "-|>", stroke: edge-stroke),
)
