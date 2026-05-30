# Terminal-native diagrams

The diagrams embedded in the engagement deck (`presentation/README.md`) are authored in
[Typst](https://typst.app) with the [Fletcher](https://typst.app/universe/package/fletcher)
graph package, rendered to PNG, and displayed inline by presenterm in the terminal.

They borrow the toolkit infrastructure in `~/tau/diagrams` (the `tokens.typ` values, the
Typst + Fletcher + CeTZ stack, the dual-render idea) but **diverge deliberately on colour**.

## Philosophy: flat, not layered

`~/tau/diagrams` uses a GitHub-Primer-derived palette where every hue is a
`(stroke, fill, ink, divider)` quad — a tinted fill plus an on-fill text colour — so a shape
reads legibly as a *card* on an arbitrary surface (light or dark document, README, web).

These diagrams have no such problem: they render **into the terminal**, on the terminal's own
background, beside terminal text. So the colour model collapses to one value per role:

- the **canvas is the terminal background** — no floating card,
- **body text is the terminal foreground**,
- **shapes are outline-only** (accent stroke + accent title, `fill: none`),
- an **accent is a single value**, used directly the way the terminal would.

The result is a *super-capable ASCII graph*: real shapes, layout, and Nerd Font glyphs, drawn
with the terminal's own palette — never chromatic fills competing with the terminal.

## Palette — `theme.typ`

Values mirror the omarchy **kanagawa** theme (`~/.config/omarchy/current/theme/kitty.conf`).
Each accent maps 1:1 onto a terminal colour slot, noted inline in the source. Re-skinning to a
different omarchy theme is a mechanical copy of that file's slots into the table — no diagram
source changes, because diagrams reference hues by name (`palette.red`), never by hex.

| Role | Value | Source |
|---|---|---|
| `surface` | `#1f1f28` | background — canvas == terminal bg |
| `ink` / `ink-muted` / `ink-subtle` | `#dcd7ba` / `#c8c093` / `#727169` | foreground tiers |
| `border` | `#54546d` | neutral outlines (containers, databases) |
| accents | `red green yellow blue violet cyan orange` | terminal colour slots / kanagawa extended |

`tokens.typ` (typography, spacing, geometry, stroke widths) is colourless and copied verbatim
from the toolkit — it carries over unchanged.

## Authoring conventions

- **One accent per entity kind.** Locked across all four Herald diagrams:
  **Herald (the focal service) = violet**, **human actors (document source, reviewer) = cyan**,
  **Azure managed services = blue**, **TAU libraries = green**, **third-party dependencies =
  neutral `border`**. Distinct at a glance, and consistent across every diagram.
- **Services are outline rectangles**; the accent is both the stroke and the title colour, body
  text stays `ink`, the divider is neutral `border`. **Databases are neutral cylinders** with a
  database glyph — shape distinguishes kind, the neutral stroke marks them as shared infra.
- **Edges are neutral** (`ink-subtle`); labels are muted italic with a `surface` label-fill so
  they punch cleanly through the line.
- **Orthogonal routing uses coordinate endpoints, not compass anchors.** For a true 90° L-turn,
  the corner waypoint must share an exact grid row/column with its neighbouring vertex — so the
  endpoints are node grid-coordinates (`(2.1, 1.3)`) and Fletcher clips to the boundary. A
  compass anchor (`<pg.west>`) resolves to a slightly different position and tilts the leg.
- **`label-side` is relative to travel direction**, not screen space: walking *south*,
  `left` = east (screen-right), `right` = west (screen-left). Mirror two symmetric L-edges by
  giving them opposite `label-side` values.

The full ingredient reference lives in the toolkit:
`~/tau/diagrams/.claude/skills/diagram-ingredients/references/edges-and-marks.md`.

## Stack

Pinned (matching the toolkit, proven against Typst 0.14): `@preview/fletcher:0.5.8`,
`@preview/cetz:0.3.4`.

## Render pipeline

presenterm decodes images with the `image` crate, which has **no SVG decoder** — so the deck
needs a raster format. The pipeline is therefore **Typst → SVG → resvg → PNG**, wired into a
`mise run render` task (`_project/presentation/mise.toml`) that loops over every `assets/*.typ`:

```bash
# from _project/presentation/ — what `mise run render` runs per diagram
typst compile --root . assets/overview.typ assets/overview.svg
resvg --zoom 3 --background "#1f1f28" assets/overview.svg assets/overview.png
```

- `--zoom 3` rasterizes at 3× for a crisp result when presenterm scales it to the slide.
- `--background "#1f1f28"` paints the kanagawa surface behind the diagram so the PNG matches the
  terminal exactly (the SVG canvas is otherwise transparent).
- Typst exports text as vector outlines, so resvg needs no font lookup.

Embed in a slide with presenterm's sizing attribute (path is relative to `presentation/`):

```markdown
![image:width:90%](assets/overview.png)
```

## Files

```
design/tokens.typ    # colourless values (verbatim from ~/tau/diagrams)
design/theme.typ     # flat kanagawa palette (this project's terminal-native convention)
design/README.md     # this file
overview.typ         # Overview-slide architecture diagram
supply-chain.typ     # dependency / TAU-isolation diagram
classification.typ   # classification workflow state graph
release.typ          # release → CDS → IL6 distribution pipeline
*.svg                # intermediate renders (one per .typ)
*.png                # embedded rasters (one per .typ)
```
