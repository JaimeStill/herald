# Go Integration

The Go server embeds compiled client assets and serves a single HTML shell. The client-side router handles all view rendering.

## Module Registration

`app/app.go` exports `NewModule(basePath)` which creates an `http.Handler` mounted at the given path (e.g., `/app`):

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
  ts, err := web.NewTemplateSet(layoutFS, viewFS, "server/layouts/*.html", "server/views", basePath, views)
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

Key points:
- Three `embed.FS`: bundled assets (`dist/*`), layout templates (`server/layouts/*`), view templates (`server/views/*`)
- Catch-all `/{path...}` route serves the HTML shell for all client paths
- Static assets served from `/dist/` via `DistServer`
- Module registered in `cmd/server/` alongside the API module

## Shell Template

`app/server/layouts/app.html`:

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

Template variables:
- `{{ .BasePath }}` — module mount path (e.g., `/app`)
- `{{ .Title }}` — page title from `ViewDef`
- `{{ .Bundle }}` — bundle name (`app` → `dist/app.js`, `dist/app.css`)

The `<base href>` tag is critical — the client-side router reads it to resolve relative paths correctly.

## View Template

`app/server/views/shell.html`:

```html
{{ define "content" }}{{ end }}
```

Empty content block — the client-side router mounts views into `#app-content` dynamically.

## Adding Navigation Links

Header navigation lives in the layout template. Add new links in the `<nav>` element:

```html
<nav>
  <a href="prompts">Prompts</a>
  <a href="new-route">New Route</a>
</nav>
```

Links use relative paths (resolved against `<base href>`). The client-side router intercepts anchor clicks for SPA navigation.
