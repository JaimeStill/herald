# 64 — Go Web App Module, Server Integration, and Dev Experience

## Context

Issue #64 is the integration layer for the web client foundation (#57). Dependencies #62 (`pkg/web/`) and #63 (client build + design + app) are merged. The client lives at `app/` with built output in `app/dist/` (`app.js`, `app.css`). The `pkg/web/` infrastructure (TemplateSet, Router, DistServer) is ready. This issue wires the Go embedding, HTML shell template, server module mounting, and Air/mise dev workflow.

## Implementation

### Step 1: HTML Shell Template — `app/server/layouts/app.html`

New file. Go HTML template serving as the SPA shell. Adapted from agent-lab's `web/app/server/layouts/app.html`:

- `<base href="{{ .BasePath }}/">` for portable URL generation
- `<link>` to `dist/{{ .Bundle }}.css`
- App header with nav links matching client routes: Documents (default `/app/`), Prompts (`/app/prompts`), Review omitted from nav (accessed via document cards)
- `<main id="app-content">` container — client router mounts views here
- `<script type="module">` for `dist/{{ .Bundle }}.js`
- No favicon/public files (Herald doesn't have them yet — can add later)

### Step 2: Shell View Template — `app/server/views/shell.html`

New file. Empty content block — client-side router takes over:
```
{{ define "content" }}{{ end }}
```

### Step 3: Go Embedding Module — `app/app.go`

New file. `package app` — follows agent-lab's `web/app/app.go` pattern:

- `//go:embed dist/*` — bundled JS/CSS
- `//go:embed server/layouts/*` — layout templates
- `//go:embed server/views/*` — view templates
- Single `ViewDef`: catch-all `/{path...}` → `shell.html`, title "Herald", bundle "app"
- `NewModule(basePath string) (*module.Module, error)`:
  1. Create `TemplateSet` from embedded layouts/views
  2. Build `web.Router` with catch-all page handler + `/dist/` file server
  3. Return `module.New(basePath, router)`
- No `public/` embed (no favicon assets yet)

Key difference from agent-lab: no `publicFS`/`publicFiles` since Herald doesn't have favicon files. The `DistServer` call uses `web.DistServer(distFS, "dist", "/dist/")` rather than agent-lab's raw `http.FileServer` — uses the existing helper from `pkg/web/static.go`.

### Step 4: Server Integration — `cmd/server/modules.go`

Modify existing file:

- Add `App *module.Module` to `Modules` struct
- In `NewModules()`: create app module via `app.NewModule("/app")`
- In `Mount()`: mount `m.App` alongside `m.API`
- Add import for `"github.com/JaimeStill/herald/app"`

### Step 5: Air Configuration — `.air.toml`

New file. Configures Air for Go hot reload:

- **Watch**: `cmd/`, `internal/`, `pkg/`, `app/dist/`, `app/server/`, `app/app.go`
- **Exclude**: `app/client/`, `app/scripts/`, `app/plugins/`, `app/node_modules/`
- **Build command**: `go build -o ./bin/server ./cmd/server`
- **Run binary**: `./bin/server`
- **Extensions**: `.go`, `.html`, `.js`, `.css`

### Step 6: Mise Tasks — `.mise.toml`

Add two tasks to existing file:

- `web:build` — `cd app && bun run build`
- `web:watch` — `cd app && bun run watch`

## Files

| File | Action |
|------|--------|
| `app/server/layouts/app.html` | Create |
| `app/server/views/shell.html` | Create |
| `app/app.go` | Create |
| `cmd/server/modules.go` | Modify |
| `.air.toml` | Create |
| `.mise.toml` | Modify |

## Validation Criteria

- `go build ./cmd/server` compiles with embedded web assets
- `go vet ./...` passes
- `GET /app/` serves HTML shell with correct `<base href>`, CSS, and JS references
- `GET /app/dist/app.js` serves bundled JavaScript
- `GET /app/dist/app.css` serves extracted global CSS
- Client-side routing works for `/app/`, `/app/prompts`, `/app/review/:id`
- Unmatched `/app/*` routes fall through to shell template (SPA fallback)
- API module continues to work at `/api/*`
- `mise run web:build` and `mise run web:watch` tasks work
- Air rebuilds Go server when `app/dist/` files change
