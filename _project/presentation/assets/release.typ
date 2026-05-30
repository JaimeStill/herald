// release — Herald's full distribution path, for the Distribution section.
//
// Terminal-native style (see design/README.md): outline-only shapes, one accent
// per entity kind. Accents:
//   actors = cyan · registries/artifacts = neutral border · CDS automation =
//   yellow · IL6 secure storage = blue (focal, heavier outline).
// (Herald-violet is intentionally absent: this diagram is about distributing
// Herald's artifacts across security domains, not the running service.)
//
// TWO repositories, each its own band, left → right across security domains.
// Artifacts / pipeline / storage sit along the top row; each band's maintainer
// is centred at the bottom of its band, beneath the shapes it triggers:
//
//   github.com/JaimeStill/herald   github.com/s2va/herald   IL6 secure env
//   ┌──────────────────────────┐   ┌──────────────────┐     ┌──────────────┐
//   │ [Service]   [Migrate]    │   │ [CDS proxy] ══════╪XDOM═╪▶[IL6 storage]│
//   │      ╲        ╱          │──▶│       ▲           │     │              │
//   │     [Maintainer]         │   │  [Maintainer]     │     │              │
//   └──────────────────────────┘   └──────────────────┘     └──────────────┘
//          (pull artifacts: one container-level edge into the proxy)
//
// The public repo publishes two artifacts (service image to GHCR, migrate CLI
// to GitHub Releases); the s2va proxy repo's pipeline pulls them, bundles +
// checksums each, stages to government-cloud storage, and submits a cross-domain
// transfer. Focal element: the transfer edge into IL6 (heavier stroke).

#import "@preview/fletcher:0.5.8" as fletcher: diagram, node, edge
#import "design/tokens.typ": tokens
#import "design/theme.typ": palette

#set page(width: auto, height: auto, margin: tokens.pad-inside-container, fill: palette.surface)
#set text(font: tokens.font, fill: palette.ink)

// This is the widest diagram (3 bands), so it is scaled down hard to fit slide
// width. Bump the type up locally (only here, not in shared tokens.typ) so
// labels stay legible after that downscale. Mirrors classification.typ.
#let scale = 1.7
#let sz-glyph = 18pt * scale
#let sz-title = tokens.size-body * scale
#let sz-label = tokens.size-label * scale

// ---- Shape helper (flat); `stroke-weight` lets the focal IL6 terminal take a
//      heavier outline ---------------------------------------------------------

#let kinded(pos, accent, glyph, title, kind, description,
             ref: none, w: auto, fill: none,
             stroke-weight: tokens.stroke-default) = node(pos,
  grid(columns: (auto, auto), column-gutter: tokens.gap-cell,
    align: (left + top, left + top),
    text(size: sz-glyph, fill: accent, glyph),
    block(width: w, stack(dir: ttb, spacing: tokens.gap-structured-text * scale,
      block(width: 100%, align(center,
        text(size: sz-title, weight: tokens.weight-bold, fill: accent, title))),
      block(width: 100%, align(center,
        text(size: sz-label, weight: tokens.weight-light, fill: palette.ink-muted,
          style: "italic", "(" + kind + ")"))),
      block(spacing: 0pt, line(length: 100%, stroke: tokens.stroke-thin + palette.border)),
      block(width: 100%, text(size: sz-label, fill: palette.ink, description)),
    )),
  ),
  shape: fletcher.shapes.rect,
  fill: fill,
  stroke: stroke-weight + accent,
  inset: tokens.pad-inside-shape,
  corner-radius: tokens.radius-shape,
  name: ref,
)

#let bandhead(pos, s, ref: none) = node(pos,
  text(size: sz-title, weight: tokens.weight-bold, fill: palette.ink, s),
  shape: fletcher.shapes.rect, fill: palette.surface-muted, stroke: none,
  inset: 4pt, name: ref)

#let edge-stroke = tokens.stroke-default + palette.ink-subtle
#let edge-stroke-emphasis = tokens.stroke-emphasis + palette.ink-muted

#let step-label(s) = text(
  size: sz-label, weight: tokens.weight-light,
  fill: palette.ink-muted, style: "italic", s,
)

// ---- Diagram -------------------------------------------------------------

#diagram(
  spacing: (4.0 * tokens.space-between-shapes, 2.7 * tokens.space-between-shapes),

  // ---- Top row: a single horizontal spine ---------------------------------
  // Service image → Migrate tool → CDS proxy → IL6, all at y=0 so the
  // "pull artifacts" and "cross-domain transfer" edges are collinear.

  // Glyph: \u{F1B2} = cube — container image.
  kinded((0, 0), palette.border, "\u{F1B2}",
    "Service image", "container · GHCR",
    [The Herald service as a container image, published to the GitHub Container Registry.],
    w: 235pt, ref: <ghcr>),

  // Glyph: \u{F1C0} = database — the migrate tool manages DB operations.
  kinded((1.25, 0), palette.border, "\u{F1C0}",
    "Migrate tool", "CLI · GitHub Releases",
    [The database migration CLI binary, published to the repository's GitHub releases.],
    w: 235pt, ref: <ghrel>),

  // Glyph: \u{F085} = cogs — automation.
  kinded((3.0, 0), palette.yellow, "\u{F085}",
    "CDS proxy pipeline", "GitHub Action",
    [Pulls the artifact and version indicated by the release tag, bundles it with a SHA-256
     checksum, stages it to CDS Send storage, and initiates a cross-domain transfer.],
    w: 270pt, ref: <proxy>),

  // Glyph: \u{F023} = lock — sealed/classified destination.
  kinded((5.1, 0), palette.blue, "\u{F023}",
    "CDS Destination", "Azure Blob Storage",
    [Artifacts arrive at the CDS receive storage container, where they are retrieved and used
     in the IL6 environment.],
    w: 235pt, ref: <il6>,
    stroke-weight: tokens.stroke-emphasis),

  // ---- Bottom row: maintainers, centred beneath their band's shapes ------

  // Glyph: \u{F007} = user. Centred under Service image + Migrate tool.
  kinded((0.625, 1.6), palette.cyan, "\u{F007}",
    "Maintainer", "release author",
    [Tags a targeted release for either the service image or the migrate CLI tool based on the
     format of the tag.],
    w: 235pt, ref: <m-pub>),

  // Centred under the CDS proxy pipeline. Same glyph; band + (kind) differ.
  kinded((3.0, 1.6), palette.cyan, "\u{F007}",
    "Maintainer", "deployment dev",
    [Pushes a tag that mirrors the release tag, triggering the CDS action workflow for the
     corresponding artifact.],
    w: 235pt, ref: <m-il4>),

  // ---- Security-domain band headers (foreground over snap:-1 fill) -------

  bandhead((0.625, -1.05), "github.com/JaimeStill/herald", ref: <pub-header>),
  bandhead((3.0, -1.05), "github.com/s2va/herald", ref: <il4-header>),
  bandhead((5.1, -1.05), "IL6 Azure", ref: <il6-header>),

  // ---- Security-domain bands (snap: -1, uniform fill) --------------------

  node(enclose: (<pub-header>, <ghcr>, <ghrel>, <m-pub>), [],
    shape: fletcher.shapes.rect, fill: palette.surface-muted, stroke: none,
    inset: 14pt, corner-radius: tokens.radius-container, snap: -1),

  node(enclose: (<il4-header>, <proxy>, <m-il4>), [],
    shape: fletcher.shapes.rect, fill: palette.surface-muted, stroke: none,
    inset: 14pt, corner-radius: tokens.radius-container, snap: -1),

  node(enclose: (<il6-header>, <il6>), [],
    shape: fletcher.shapes.rect, fill: palette.surface-muted, stroke: none,
    inset: 14pt, corner-radius: tokens.radius-container, snap: -1),

  // ---- Trigger edges (maintainer → artifact): leave the maintainer's side
  //      centre, then turn 90° straight up into the artifact. Coordinate
  //      waypoints share the artifact's x and the maintainer's y for true
  //      right-angle legs. --------------------------------------------------

  edge(<m-pub.west>, (0, 1.6), <ghcr.south>, "->", step-label("tag v*"),
    label-pos: 0.7, label-side: center, label-sep: tokens.label-sep,
    label-fill: palette.surface, stroke: edge-stroke),

  edge(<m-pub.east>, (1.25, 1.6), <ghrel.south>, "->", step-label("tag migrate-v*"),
    label-pos: 0.7, label-side: center, label-sep: tokens.label-sep,
    label-fill: palette.surface, stroke: edge-stroke),

  // Deployment dev → proxy: vertical trigger.
  edge(<m-il4.north>, <proxy.south>, "->", step-label("tag: [v* | migrate-v*]"),
    label-pos: 0.5, label-side: center, label-sep: tokens.label-sep,
    label-fill: palette.surface, stroke: edge-stroke),

  // ---- Spine: Service → Migrate (plain, no mark/label) → CDS proxy --------

  edge(<ghcr.east>, <ghrel.west>, "-", stroke: edge-stroke),

  edge(<ghrel.east>, <proxy.west>, "->", step-label("pull artifact"),
    label-pos: 0.5, label-side: center, label-sep: tokens.label-sep,
    label-fill: palette.surface, stroke: edge-stroke),

  // ---- IL4 → IL6 boundary crossing (FOCAL), collinear with the spine -----

  edge(<proxy.east>, <il6.west>, "->", step-label("cross-domain transfer"),
    label-pos: 0.5, label-side: center, label-sep: tokens.label-sep,
    label-fill: palette.surface, stroke: edge-stroke-emphasis),
)
