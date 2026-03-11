# Build System

Native `Bun.build()` API — no Vite or Webpack. Single entry point produces fixed-name outputs for stable `go:embed` globs.

## Commands

```bash
bun run build    # Single build: client/app.ts → dist/app.js + dist/app.css
bun run watch    # Watch client/ for .ts/.css changes, rebuild with 150ms debounce
```

## Build Script

`app/scripts/build.ts`:

```typescript
import { litCSSModulePlugin } from "../plugins/css-modules";

const result = await Bun.build({
    entrypoints: ["client/app.ts"],
    outdir: "dist",
    naming: "app.[ext]",
    plugins: [litCSSModulePlugin],
    minify: false,
});
```

- Single entry point: `client/app.ts`
- Output: `dist/app.js` + `dist/app.css` (fixed names)
- CSS modules plugin transforms `*.module.css` → `CSSStyleSheet`
- Non-module CSS extracted automatically to `app.css`

## CSS Modules Plugin

`app/plugins/css-modules.ts` intercepts `*.module.css` imports:

1. `onResolve` — redirects `*.module.css` to the `lit-css` namespace
2. `onLoad` — reads the CSS file, wraps content in `new CSSStyleSheet()` + `replaceSync()`

The result is a JavaScript module that exports a `CSSStyleSheet` object, which Lit accepts directly in `static styles`.

Non-module CSS (`import './design/index.css'`) is never intercepted — it flows to Bun's default pipeline and gets extracted to `dist/app.css`.

## Watch Script

`app/scripts/watch.ts` uses Node's `fs.watch` with a 150ms debounce. Triggers a full rebuild on any `.ts` or `.css` change in `client/`.

## Dev Workflow

Two terminals, clean separation of concerns:

| Terminal | Command         | Watches                    | Rebuilds                   |
| -------- | --------------- | -------------------------- | -------------------------- |
| 1        | `bun run watch` | `app/client/**/*.{ts,css}` | Client assets → `dist/`    |
| 2        | `air`           | Go + `dist/` + templates   | Go binary → restart server |

Air's `.air.toml` configuration:

- **Includes**: `cmd/`, `internal/`, `pkg/`, `app/` (Go files, templates, dist output)
- **Excludes**: `app/client`, `app/scripts`, `app/plugins`, `app/node_modules`
- **Extensions**: `.go`, `.html`, `.js`, `.css`

This separation means Bun handles TypeScript/CSS compilation and Air handles Go compilation + server restart. Changes to client source trigger Bun → writes to `dist/` → Air detects dist change → restarts server.

## Path Aliases

Defined in `tsconfig.json` paths and mirrored in the Bun build plugin:

```json
{
    "compilerOptions": {
        "paths": {
            "@core": ["./client/core/index.ts"],
            "@core/*": ["./client/core/*"],
            "@design/*": ["./client/design/*"],
            "@domains/*": ["./client/domains/*"],
            "@styles/*": ["./client/design/styles/*"],
            "@ui/*": ["./client/ui/*"]
        }
    }
}
```

See SKILL.md **Import Convention** for usage patterns.

## TypeScript Configuration

Key settings in `tsconfig.json`:

- `target: "es2024"` — modern JavaScript output
- `experimentalDecorators: true` — required for Lit decorators
- `useDefineForClassFields: false` — required for Lit reactivity (class fields must use `[Set]` semantics)
- `moduleResolution: "bundler"` — matches Bun's resolution strategy
