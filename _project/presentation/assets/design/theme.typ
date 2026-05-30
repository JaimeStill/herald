// ==============================================================
// Terminal-native theme — kanagawa (omarchy current).
//
// These diagrams render INTO the terminal, on the terminal's own
// background, embedded in a presenterm deck. So the colour model is
// FLAT, not the layered Primer model (per-hue stroke/fill/ink/divider
// quads tuned to sit legibly on an arbitrary surface). Here:
//
//   - the canvas IS the terminal background (no floating card),
//   - body text IS the terminal foreground,
//   - shapes are outline-only (accent stroke + accent title, fill:none),
//   - an accent is a single value, used directly as the terminal would.
//
// Think: a super-capable ASCII graph drawn with the terminal's own
// 16-colour palette — real shapes, layout, and Nerd Font glyphs, but
// no chromatic fills competing with the terminal.
//
// Values mirror the omarchy kanagawa palette
// (~/.config/omarchy/current/theme/kitty.conf). Re-skinning to another
// omarchy theme is a mechanical copy of that file's colour slots into
// the table below — accents map 1:1 onto ANSI slots, noted per line.
// ==============================================================

#import "tokens.typ": tokens

#let palette = (
  // ---- surfaces (kanagawa sumiInk) ----
  surface:        rgb("#1f1f28"),  // background — canvas == terminal bg
  surface-muted:  rgb("#2a2a37"),  // sumiInk2 — subtle grouping panel
  surface-raised: rgb("#16161d"),  // sumiInk0

  // ---- ink (kanagawa fuji) ----
  ink:            rgb("#dcd7ba"),  // foreground — body text
  ink-muted:      rgb("#c8c093"),  // oldWhite — edge labels, secondary text
  ink-subtle:     rgb("#727169"),  // fujiGray — de-emphasized

  // ---- borders ----
  border:         rgb("#54546d"),  // sumiInk4 — neutral outlines (containers, DBs)
  border-muted:   rgb("#363646"),  // sumiInk3

  // ---- accents: single value each, used as stroke + title ----
  // Entity mapping is LOCKED across all four Herald diagrams (see design/README.md):
  violet:  rgb("#957fb8"),  // oniViolet    (ANSI color5)            — Herald (focal service)
  cyan:    rgb("#7aa89f"),  // waveAqua2    (ANSI color14, bright)   — human actors (source, reviewer)
  green:   rgb("#98bb6c"),  // springGreen  (ANSI color10, bright)   — TAU libraries
  blue:    rgb("#7e9cd8"),  // crystalBlue  (ANSI color4)            — Azure managed services
  // available, unassigned — reach for these only if a new entity kind appears:
  red:     rgb("#c34043"),  // autumnRed    (ANSI color1)
  yellow:  rgb("#e6c384"),  // carpYellow   (ANSI color11, bright)
  orange:  rgb("#ffa066"),  // surimiOrange (kanagawa extended)
  // (third-party dependencies use the neutral `border` above — not an accent.)
)
