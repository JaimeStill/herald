# CSS Architecture

## Cascade Layers

Herald uses four layers with explicit precedence. Tokens are lowest priority (easily overridden), app is highest.

```css
/* design/index.css */
@layer tokens, reset, theme, app;

@import url(./core/tokens.css);
@import url(./core/reset.css);
@import url(./core/theme.css);

@import url(./app/app.css);
```

`app.css` is in the `app` layer (highest priority) and handles the application shell layout.

## Design Tokens

CSS custom properties in `:root` with light/dark mode via `prefers-color-scheme`:

**Spacing** — 0.25rem base unit:
`--space-1` (0.25rem), `--space-2` (0.5rem), `--space-3` (0.75rem), `--space-4` (1rem), `--space-5` (1.25rem), `--space-6` (1.5rem), `--space-8` (2rem), `--space-10` (2.5rem), `--space-12` (3rem), `--space-16` (4rem)

**Typography**:
`--text-xs` (0.75rem), `--text-sm` (0.875rem), `--text-base` (1rem), `--text-lg` (1.125rem), `--text-xl` (1.25rem), `--text-2xl` (1.5rem), `--text-3xl` (1.875rem), `--text-4xl` (2.25rem)

**Fonts**:
`--font-sans` (system-ui stack), `--font-mono` (ui-monospace stack)

**Colors** (dark mode default, light mode via media query):
- Backgrounds: `--bg`, `--bg-1`, `--bg-2`
- Text: `--color`, `--color-1`, `--color-2`
- Border: `--divider`
- Semantic: `--blue`, `--green`, `--red`, `--yellow`, `--orange` — each has a `-bg` variant for backgrounds

**Radii**: `--radius-sm` (0.25rem), `--radius-md` (0.5rem), `--radius-lg` (0.75rem)

**Shadows**: `--shadow-sm`, `--shadow-md`, `--shadow-lg`

Tokens penetrate shadow DOM boundaries because CSS custom properties inherit naturally. This is how component styles access the design system without workarounds.

## Component Styles via CSS Modules

Component CSS uses the `*.module.css` naming convention. The Bun plugin at `app/plugins/css-modules.ts` transforms these into `CSSStyleSheet` objects that Lit accepts directly in `static styles`:

```typescript
import styles from './documents-view.module.css';

static styles = styles;
```

This produces native `CSSStyleSheet` objects — no `unsafeCSS()` wrapper needed (unlike agent-lab's `?inline` pattern).

The TypeScript declaration at `app/client/css.d.ts` enables the import:

```typescript
declare module '*.module.css' {
  const styles: CSSStyleSheet;
  export default styles;
}
```

### Component CSS Patterns

Use `:host` for component-level layout and design tokens for all values:

```css
:host {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
  padding: var(--space-6);
}

.card {
  background: var(--bg-1);
  border: 1px solid var(--divider);
  border-radius: var(--radius-md);
  padding: var(--space-4);
}

h3 { color: var(--color); }
p { color: var(--color-1); }
```

## Shared Component Styles

Reusable CSS modules in `app/client/design/styles/` imported via `@styles/*`. Components add these to `static styles` arrays alongside their own `*.module.css`.

| Module | Class | Purpose |
|--------|-------|---------|
| `badge.module.css` | `.badge` + status variants (`.pending`, `.review`, `.complete`, etc.) | Status badges with semantic colors |
| `buttons.module.css` | `.btn` + color variants (`.btn-blue`, `.btn-green`, `.btn-red`, `.btn-yellow`, `.btn-muted`) | Button base with semantic color overlays |
| `cards.module.css` | `.card` | Flex column container with gap, padding, border, radius, transition |
| `inputs.module.css` | `.input` | Text inputs, selects, and textareas with focus/disabled states |
| `labels.module.css` | `.label` | Uppercase monospace section labels (form field labels, section headers) |

Usage pattern:

```typescript
import badgeStyles from "@styles/badge.module.css";
import buttonStyles from "@styles/buttons.module.css";
import cardStyles from "@styles/cards.module.css";
import styles from "./document-card.module.css";

static styles = [buttonStyles, badgeStyles, cardStyles, styles];
```

Shared styles provide base appearance. Component CSS retains layout-specific overrides (e.g., `.search-input { flex: 1; min-width: 12rem; }`). Button color variants compose with the `.btn` base: `class="btn btn-blue"`.

## Global CSS

Side-effect imports (no module suffix) flow through Bun's default pipeline and are extracted to `dist/app.css`:

```typescript
// app.ts entry point
import './design/index.css';
```

Only the entry point imports global CSS. Components never import global stylesheets.

## App-Shell Scroll Architecture

Body fills viewport and never scrolls. Views manage their own scroll regions. This prevents competing scrollbars and gives each view full control over its layout.

```css
body {
  display: flex;
  flex-direction: column;
  height: 100svh;
  margin: 0;
  overflow: hidden;
}

#app-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
  overflow: hidden;
}

#app-content > * {
  flex: 1;
  min-height: 0;
}
```

View pattern for scrollable content — the critical pieces are `flex: 1`, `min-height: 0`, and `overflow-y: auto` on the scrollable child:

```css
:host {
  display: flex;
  flex-direction: column;
}

.scrollable {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
}
```

Without `min-height: 0` on flex children, content overflows instead of scrolling. This is the most common CSS bug in the application.
