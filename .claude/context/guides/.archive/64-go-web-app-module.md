# 64 — Go Web App Module, Server Integration, and Dev Experience

## Problem Context

The web client foundation (#57) has two completed dependencies: `pkg/web/` (#62) provides Go-side template/asset serving infrastructure, and the client build system (#63) delivers a Lit 3.x SPA at `app/` with bundled output in `app/dist/`. This issue wires the two together — embedding built assets in the Go binary, serving the SPA shell via templates, mounting the app module alongside the API, and configuring the two-terminal dev workflow (Bun watch + Air).

## Architecture Approach

Follow agent-lab's `web/app/app.go` pattern directly. The `app/` directory becomes a Go package (`package app`) that embeds its own `dist/` and `server/` subdirectories. A single `NewModule(basePath)` factory creates a `*module.Module` that the server mounts at `/app`. All routes under `/app/*` fall through to the SPA shell template — the client-side History API router handles view switching.

Key adaptation from agent-lab: no `public/` embed (no favicon assets yet), and use `web.DistServer()` helper from `pkg/web/static.go` instead of raw `http.FileServer`.

## Implementation

### Step 1: Create `app/server/layouts/app.html`

```html
<!DOCTYPE html>
<html lang="en">

<head>
  <base href="{{ .BasePath }}/">
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>{{ .Title }} - Herald</title>
  <link rel="stylesheet" href="dist/{{ .Bundle }}.css">
</head>

<body>
  <header class="app-header">
    <a href="" class="brand">Herald</a>
    <nav>
      <a href="">Documents</a>
      <a href="prompts">Prompts</a>
    </nav>
  </header>
  <main id="app-content">
    {{ block "content" . }}{{ end }}
  </main>

  <script type="module" src="dist/{{ .Bundle }}.js"></script>
</body>

</html>
```

Nav links use relative URLs (resolved against `<base href>`). Review view is omitted from nav — accessed via document cards.

### Step 2: Create `app/server/views/shell.html`

```
{{ define "content" }}{{ end }}
```

Empty content block. The client-side router mounts Lit components into `#app-content`.

### Step 3: Create `app/app.go`

```go
package app

import (
	"embed"
	"net/http"

	"github.com/JaimeStill/herald/pkg/module"
	"github.com/JaimeStill/herald/pkg/web"
)

//go:embed dist/*
var distFS embed.FS

//go:embed server/layouts/*
var layoutFS embed.FS

//go:embed server/views/*
var viewFS embed.FS

var views = []web.ViewDef{
	{Route: "/{path...}", Template: "shell.html", Title: "Herald", Bundle: "app"},
}

func NewModule(basePath string) (*module.Module, error) {
	ts, err := web.NewTemplateSet(
		layoutFS,
		viewFS,
		"server/layouts/*.html",
		"server/views",
		basePath,
		views,
	)
	if err != nil {
		return nil, err
	}

	router := buildRouter(ts)
	return module.New(basePath, router), nil
}

func buildRouter(ts *web.TemplateSet) http.Handler {
	r := web.NewRouter()

	r.Handle("GET /dist/", web.DistServer(distFS, "dist", "/dist/"))
	r.SetFallback(ts.PageHandler("app.html", views[0]))

	return r
}
```

Key decisions:
- `/dist/` route registered first as a concrete match, fallback catches everything else (SPA catch-all)
- Uses `web.DistServer` for static asset serving with proper prefix stripping
- Uses `r.SetFallback` instead of registering `/{path...}` — the `web.Router` fallback mechanism handles unmatched routes cleanly

### Step 4: Modify `cmd/server/modules.go`

Add `app` import and `App` field to `Modules` struct. Create and mount the app module.

```go
package main

import (
	"encoding/json"
	"net/http"

	"github.com/JaimeStill/herald/app"
	"github.com/JaimeStill/herald/internal/api"
	"github.com/JaimeStill/herald/internal/config"
	"github.com/JaimeStill/herald/internal/infrastructure"
	"github.com/JaimeStill/herald/pkg/module"
)

type Modules struct {
	API *module.Module
	App *module.Module
}

func NewModules(infra *infrastructure.Infrastructure, cfg *config.Config) (*Modules, error) {
	apiModule, err := api.NewModule(cfg, infra)
	if err != nil {
		return nil, err
	}

	appModule, err := app.NewModule("/app")
	if err != nil {
		return nil, err
	}

	return &Modules{
		API: apiModule,
		App: appModule,
	}, nil
}

func (m *Modules) Mount(router *module.Router) {
	router.Mount(m.API)
	router.Mount(m.App)
}
```

The `buildRouter` function is unchanged — omitted for brevity.

### Step 5: Create `.air.toml`

**Pre-requisite:** Install Air as a global Go binary:

```bash
go install github.com/air-verse/air@latest
```

```toml
root = "."
tmp_dir = "tmp"

[build]
  bin = "./bin/server"
  cmd = "go build -o ./bin/server ./cmd/server"
  delay = 1000
  exclude_dir = ["bin", "tmp", "node_modules", "app/client", "app/scripts", "app/plugins", "app/node_modules", "tests", "_project", ".claude"]
  exclude_regex = ["_test\\.go$"]
  include_dir = ["cmd", "internal", "pkg", "app"]
  include_ext = ["go", "html", "js", "css"]
  kill_delay = 500
  send_interrupt = true

[log]
  time = false

[misc]
  clean_on_exit = true
```

Watch scope includes `app/` (catches `app.go`, `server/` templates, and `dist/` output). Excludes client source directories since Bun handles those — only the build output (`dist/`) triggers a Go rebuild.

### Step 6: Modify `.mise.toml`

Add two tasks at the end of the file:

```toml
[tasks."web:build"]
description = "Build the web client"
run = "cd app && bun run build"

[tasks."web:watch"]
description = "Watch and rebuild the web client"
run = "cd app && bun run watch"
```

## Validation Criteria

- [ ] `go build ./cmd/server` compiles with embedded web assets
- [ ] `go vet ./...` passes
- [ ] `GET /app/` serves HTML shell with `<base href="/app/">`, CSS link, JS script
- [ ] `GET /app/dist/app.js` serves bundled JavaScript with correct MIME type
- [ ] `GET /app/dist/app.css` serves extracted global CSS
- [ ] Client-side routing works for `/app/`, `/app/prompts`, `/app/review/test-id`
- [ ] Unmatched `/app/foo` routes fall through to shell template (SPA fallback)
- [ ] API module continues to work at `/api/*` alongside web app at `/app/*`
- [ ] `mise run web:build` and `mise run web:watch` tasks work
- [ ] Air rebuilds Go server when `app/dist/` files change
