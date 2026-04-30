// core/release — Herald's full distribution path (stakeholder voice).
//
// Three security-domain bands, horizontal flow:
//   PUBLIC ────► IL4 TRANSIT ────► IL6 SECURE
//
// A maintainer pushes a tag in the Herald repo to trigger the public
// release; a matching tag is then manually mirrored in the proxy repo
// to initiate the cross-domain transfer. The chain terminates at IL6
// secure storage. Future state: an IL6-side release workflow will
// complete the automated deployment update once GitHub Actions are
// available on the IL6 GitHub instance.
//
// Focal element: the `cross-domain transfer` edge from the proxy
// pipeline into IL6 secure storage. Heavier stroke + the IL6 destination
// stroke-emphasis carry the diagram's visual centre of gravity.
//
// Layout:
//   [M-pub]                          [M-il4]
//      │ push tag (v*)                  │ manually mirror tag
//      ▼                                ▼
//   ┌─ Public ───────────────┐  ┌─ IL4 ─┐  ┌─ IL6 ─┐
//   │ [Public pipeline] ─image─▶ [GHCR] │─▶ [CDS pipeline] ─XDOM─▶ [IL6 storage]
//   │      │                          │ │
//   │      └─notes─▶ [Release page]   │ │
//   └────────────────────────────────┘  └────────┘  └──────────────┘
//
// `kinded` is extended with an optional `stroke-weight` parameter for
// the IL6 destination's heavier outline. Backwards-compatible.

#import "@preview/fletcher:0.5.8" as fletcher: diagram, node, edge
#import "../design/tokens.typ": tokens
#import "../design/theme.typ": palette

#set page(width: auto, height: auto, margin: tokens.pad-inside-container, fill: palette.surface)
#set text(font: tokens.font, fill: palette.ink)

// ---- Shape helper --------------------------------------------------------

// `kinded` — body shape with glyph + stacked(title / (kind) / divider /
// description). Title and (kind) centred within the body's natural width;
// description stays left-aligned. Optional `extras` content appended below.
//
// Extension (B3d): `stroke-weight` lets a focal terminal take a heavier
// outline (e.g., IL6 secure storage). Defaults to `tokens.stroke-default`
// for backwards compatibility with prior diagrams.
#let kinded(pos, hue, glyph, title, kind, description,
             extras: none, ref: none,
             stroke-weight: tokens.stroke-default) = node(pos,
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
  stroke: stroke-weight + hue.stroke,
  inset: tokens.pad-inside-shape,
  corner-radius: tokens.radius-shape,
  name: ref,
)

// ---- Edge styling --------------------------------------------------------

#let edge-stroke = tokens.stroke-default + palette.green.stroke
#let edge-stroke-emphasis = tokens.stroke-emphasis + palette.green.stroke

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

  // ---- Maintainers (orange = actor) --------------------------------------

  // Glyph: \u{F007} = user. Same identity on both sides; visual repetition
  // makes the dual-trigger explicit.
  kinded((-3, -2.0), palette.orange, "\u{F007}",
    "Maintainer", "release author",
    [Tags a Herald version \
     to release.],
    ref: <m-pub>,
  ),

  kinded((1, -2.0), palette.orange, "\u{F007}",
    "Maintainer", "release author",
    [Pushes a matching tag \
     in the proxy repository \
     to initiate the cross- \
     domain transfer. This is \
     a deliberate human gate.],
    ref: <m-il4>,
  ),

  // ---- Public-side pipeline + terminals (purple state, coral terminals) --

  // Glyph: \u{F085} = cogs — automation.
  kinded((-3, 0), palette.purple, "\u{F085}",
    "Public release pipeline", "automation",
    [Triggered by a version- \
     tag push. Builds the \
     container image and \
     publishes release notes \
     from the CHANGELOG.],
    ref: <pub>,
  ),

  // Glyph: \u{F49E} = box-open — image as a deliverable package.
  kinded((-1, 0), palette.coral, "\u{F49E}",
    "GitHub Container Registry", "container registry",
    [Public destination for \
     the tagged Herald image. \
     Bridge between the public \
     release and the cross- \
     domain transfer.],
    ref: <ghcr>,
  ),

  // Glyph: \u{F1EA} = newspaper — release notes / publication.
  kinded((-3, 1), palette.coral, "\u{F1EA}",
    "GitHub Releases page", "release notes",
    [Public release page \
     rendered from the \
     CHANGELOG entry for \
     the tagged version.],
    ref: <release>,
  ),

  // ---- IL4 transit pipeline ----------------------------------------------

  // Glyph: \u{F085} = cogs — automation. Same as public pipeline; band
  // membership and description differentiate.
  kinded((1, 0), palette.purple, "\u{F085}",
    "Cross-domain release pipeline", "automation",
    [Pulls the published \
     image, packages it as \
     a checksummed bundle, \
     stages it to government \
     cloud storage, and \
     submits it to the \
     cross-domain transfer \
     service.],
    ref: <cds>,
  ),

  // ---- IL6 destination (focal terminal, stroke-emphasis) -----------------

  // Glyph: \u{F023} = lock — sealed/classified destination.
  kinded((3, 0), palette.coral, "\u{F023}",
    "IL6 secure storage", "classified destination",
    [Landing zone on the \
     secure-environment side \
     after the cross-domain \
     transfer is delivered. \
     Operators on the secure \
     side import the image \
     into the secure \
     environment from here.],
    ref: <il6>,
    stroke-weight: tokens.stroke-emphasis,
  ),

  // ---- Security-domain band headers (foreground nodes) -------------------

  // Each band gets a labelled header that names the repository or
  // landing zone. Fill matches the band's surface-muted so the trigger
  // arrows visually pass behind the header text rather than striking
  // through it. Two-node container pattern from B3a: header node at
  // default layer + enclose container at snap=-1.

  node((-2, -1),
    text(size: tokens.size-body, weight: tokens.weight-bold, fill: palette.ink,
         "github.com/JaimeStill/herald"),
    shape: fletcher.shapes.rect,
    fill: palette.surface-muted,
    stroke: none,
    inset: 4pt,
    name: <pub-header>,
  ),

  node((1, -1),
    text(size: tokens.size-body, weight: tokens.weight-bold, fill: palette.ink,
         "IL4 GitHub Proxy Repo"),
    shape: fletcher.shapes.rect,
    fill: palette.surface-muted,
    stroke: none,
    inset: 4pt,
    name: <il4-header>,
  ),

  node((3, -1),
    text(size: tokens.size-body, weight: tokens.weight-bold, fill: palette.ink,
         "Azure Data Transfer Landing Zone"),
    shape: fletcher.shapes.rect,
    fill: palette.surface-muted,
    stroke: none,
    inset: 4pt,
    name: <il6-header>,
  ),

  // ---- Security-domain bands (snap: -1, uniform fill) --------------------

  // Public band — encloses header + pipeline + GHCR + release page.
  node(
    enclose: (<pub-header>, <pub>, <ghcr>, <release>),
    [],
    shape: fletcher.shapes.rect,
    fill: palette.surface-muted,
    stroke: none,
    inset: 14pt,
    corner-radius: tokens.radius-container,
    snap: -1,
  ),

  // IL4 transit band — encloses header + CDS pipeline.
  node(
    enclose: (<il4-header>, <cds>),
    [],
    shape: fletcher.shapes.rect,
    fill: palette.surface-muted,
    stroke: none,
    inset: 14pt,
    corner-radius: tokens.radius-container,
    snap: -1,
  ),

  // IL6 band — encloses header + IL6 storage.
  node(
    enclose: (<il6-header>, <il6>),
    [],
    shape: fletcher.shapes.rect,
    fill: palette.surface-muted,
    stroke: none,
    inset: 14pt,
    corner-radius: tokens.radius-container,
    snap: -1,
  ),

  // ---- Trigger edges (maintainers → pipelines) ---------------------------

  // M-pub → Public pipeline: automated trigger.
  edge(<m-pub.south>, <pub.north>, "->",
    step-label("push tag (v*)"),
    label-pos: 0.3, label-side: center,
    label-sep: tokens.label-sep,
    label-fill: palette.surface,
    stroke: edge-stroke,
  ),

  // M-il4 → CDS pipeline: deliberate human gate.
  edge(<m-il4.south>, <cds.north>, "->",
    step-label("manually mirror tag"),
    label-pos: 0.3, label-side: center,
    label-sep: tokens.label-sep,
    label-fill: palette.surface,
    stroke: edge-stroke,
  ),

  // ---- Public-side fan-out -----------------------------------------------

  // Public pipeline → GHCR: image artifact.
  edge(<pub.east>, <ghcr.west>, "->",
    step-label("publish image"),
    label-pos: 0.5, label-side: center,
    label-sep: tokens.label-sep,
    label-fill: palette.surface,
    stroke: edge-stroke,
  ),

  // Public pipeline → Release page: notes from CHANGELOG.
  edge(<pub.south>, <release.north>, "->",
    step-label("publish notes from CHANGELOG"),
    label-pos: 0.5, label-side: center,
    label-sep: tokens.label-sep,
    label-fill: palette.surface,
    stroke: edge-stroke,
  ),

  // ---- Public/IL4 bridge -------------------------------------------------

  // GHCR → CDS pipeline: pull published image (cross-band).
  edge(<ghcr.east>, <cds.west>, "->",
    step-label("pull image"),
    label-pos: 0.5, label-side: center,
    label-sep: tokens.label-sep,
    label-fill: palette.surface,
    stroke: edge-stroke,
  ),

  // ---- IL4/IL6 boundary crossing (FOCAL) ---------------------------------

  // CDS pipeline → IL6 storage: the consequential cross-domain transfer.
  // Heavier stroke pulls the eye to the boundary crossing.
  edge(<cds.east>, <il6.west>, "->",
    step-label("cross-domain transfer"),
    label-pos: 0.5, label-side: center,
    label-sep: tokens.label-sep,
    label-fill: palette.surface,
    stroke: edge-stroke-emphasis,
  ),
)
