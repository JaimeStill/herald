// core/readme — Herald architecture overview (stakeholder voice).
//
// Architecture refactor (iteration 8): edge labels migrate into shape bodies.
// Each shape carries title + (kind) + divider + short description that
// articulates its role relative to Herald (the focal node). Edges become
// connectors only — direction matters, but the *what* of each relationship
// lives in the shape body where it has room and context.
//
// The four Azure managed services live inside an `enclose:` container whose
// header expresses the shared identity story (user-managed identity for
// service-to-service auth). Two edges leave Herald: one to the container
// header (bundled "uses these managed services"), one to Container Apps
// inside (specific "deployment target"). Container Apps sits at the far
// right of the container so the specific edge does not cross the header.
//
// Layout:
//   [Document source]                                    [Reviewer]
//                              [Herald]
//   ┌─── Azure Managed Services ──────────────────────────────────┐
//   │ [AI Foundry] [PostgreSQL] [Blob Storage]   [Container Apps] │
//   └─────────────────────────────────────────────────────────────┘
//
// Shape pattern: icon-left + stacked(title / (kind) / divider / description).
// Container pattern per `~/tau/diagrams/.claude/skills/diagram-ingredients/
// references/labels-and-encapsulation.md`.

#import "@preview/fletcher:0.5.8" as fletcher: diagram, node, edge
#import "../design/tokens.typ": tokens
#import "../design/theme.typ": palette

#set page(width: auto, height: auto, margin: tokens.pad-inside-container, fill: palette.surface)
#set text(font: tokens.font, fill: palette.ink)

// ---- Shape helper --------------------------------------------------------

// `kinded` — body shape with glyph + stacked(title / (kind) / divider /
// description). Title and (kind) are centred within the body's natural
// width; description stays left-aligned. Optional `extras` content appended
// below the description.
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

// ---- Edges ---------------------------------------------------------------

#let edge-stroke = tokens.stroke-default + palette.green.stroke

// ---- Diagram -------------------------------------------------------------

#diagram(
  spacing: (3 * tokens.space-between-shapes, 2.5 * tokens.space-between-shapes),

  // ---- Top row: actors ----------------------------------------------------

  // Glyph: \u{F007} = user (Font Awesome).
  kinded((-0.8, 0.5), palette.orange, "\u{F007}",
    "Document source", "upstream",
    [Submits documents \
     for classification.],
    ref: <doc>,
  ),

  kinded((0.8, 0.5), palette.orange, "\u{F007}",
    "Reviewer", "human",
    [Validates classifications \
     produced by Herald.],
    ref: <rev>,
  ),

  // ---- Centre: Herald (focal) --------------------------------------------

  // Glyph: \u{F0AC} = globe — generic service identity.
  // Microsoft Entra ID and Azure Container Registry both surface in the
  // description body — no separate identity pill, since auth and image
  // source share the same descriptive treatment.
  kinded((0, 1), palette.blue, "\u{F0AC}",
    "Herald", "service",
    [Classifies security markings on \
     DoD documents. \
      \
     Users sign in \
     via Microsoft Entra ID. \
      \
     Go binary deployed as a container; \
     image sourced from \
     Azure Container Registry.],
    ref: <h>,
  ),

  // ---- Bottom: Azure Managed Services container --------------------------

  // Inner nodes — each named so the container can enclose them and edges
  // can address them across the boundary. Order left→right:
  // AI Foundry, PostgreSQL, Blob Storage, Container Apps. Container Apps
  // sits at the far right so the specific deployment edge from Herald does
  // not cross the container header (anchored top-left).

  // Glyph: \u{F0C2} = cloud — AI service compute.
  kinded((-1, 3), palette.coral, "\u{F0C2}",
    "Azure AI Foundry", "AI service",
    [Per-page and document-level \
     vision inference.],
    ref: <ai>,
  ),
  // Glyph: \u{F1C0} = database.
  kinded((-0.33, 3), palette.coral, "\u{F1C0}",
    "Azure PostgreSQL", "database",
    [Document metadata and \
     classification records.],
    ref: <pg>,
  ),
  // Glyph: \u{F0A0} = hard-drive — object storage.
  kinded((0.33, 3), palette.coral, "\u{F0A0}",
    "Azure Blob Storage", "object storage",
    [Document blob storage.],
    ref: <blob>,
  ),
  // Glyph: \u{F1B2} = cube — runtime/deployment.
  kinded((1, 3), palette.coral, "\u{F1B2}",
    "Azure Container Apps", "runtime",
    [Herald's deployment host.],
    ref: <aca>,
  ),

  // Container header — a separate node placed at y=3 (above the inner
  // shapes at y=4). Carries the cloud glyph, title, and the UMI line. By
  // being its own node at default layer, it stays in the foreground above
  // the container's `snap: -1` fill rather than being painted behind the
  // inner shapes (as a body-on-the-container would be).
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
    fill: none,
    stroke: none,
    inset: 0pt,
    name: <azure-header>,
  ),

  // Container background — encloses the header AND the four inner shapes,
  // so its bounding box spans both. Fill only; no border; `snap: -1` so it
  // renders behind everything (including the header above the shapes).
  node(
    enclose: (<azure-header>, <ai>, <pg>, <blob>, <aca>),
    name: <azure>,
    [],
    shape: fletcher.shapes.rect,
    fill: palette.surface-muted,
    stroke: none,
    inset: tokens.pad-inside-container,
    corner-radius: tokens.radius-container,
    snap: -1,
  ),

  // ---- Edges (connectors only, no labels) --------------------------------

  // Document source → Herald (90deg waypoint: down from Doc source's bottom,
  // then right into Herald's west anchor).
  edge(<doc.south>, (-0.8, 1), <h.west>, "->", stroke: edge-stroke),

  // Reviewer ↔ Herald (single bidirectional edge, 90deg waypoint: down from
  // Reviewer's bottom, then left into Herald's east anchor).
  edge(<rev.south>, (0.8, 1), <h.east>, "<->", stroke: edge-stroke),

  // Herald → Azure Managed Services container (bundled "uses these services").
  edge(<h.south>, <azure>, "->", stroke: edge-stroke),

  // Herald → Container Apps (specific "deployment target"). Distinct mark
  // `-O` (large open circle) reads as association / dock — semantically
  // matched to "Herald is deployed on ACA" — and visually separates this
  // edge from the standard `->` data-flow edges. Destination is ACA's
  // `north` anchor so the circle lands centred on ACA's top edge.
  edge(<h.south-east>, <aca.north>, "-O", stroke: edge-stroke),
)
