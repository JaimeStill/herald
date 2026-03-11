# 118 — Inject auth config into web app HTML template

## Problem Context

Sub-issue 1 of Objective #99 (Web Client MSAL.js Integration). The Lit SPA needs MSAL configuration (tenant ID, client ID, redirect URI, authority) available synchronously at page load before any JS executes. The Go server has all auth config in `pkg/auth.Config`, but `app.NewModule` currently receives only a `basePath` string and the `ViewData.Data` field is always empty.

## Architecture Approach

Define a browser-safe `ClientAuthConfig` struct in the `app` package containing only fields safe for client exposure (no `ClientSecret`). The Go template uses `{{ if .Data }}` to conditionally render a `<script id="herald-config" type="application/json">` tag. When auth mode is `none`, `Data` is nil and the block is skipped. JSON is wrapped in `template.JS` to prevent `html/template` from escaping it inside the `<script>` tag. A new `PageHandlerWithData` method on `TemplateSet` populates `ViewData.Data` without modifying the existing `PageHandler`.

## Implementation

### Step 1: Add `PageHandlerWithData` to `pkg/web/views.go`

Add this method after the existing `PageHandler`:

```go
func (ts *TemplateSet) PageHandlerWithData(layout string, view ViewDef, data any) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		d := ViewData{
			Title:    view.Title,
			Bundle:   view.Bundle,
			BasePath: ts.basePath,
			Data:     data,
		}
		if err := ts.Render(w, layout, view.Template, d); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}
```

### Step 2: Update `app/app.go` — Add `ClientAuthConfig` and update `NewModule`

Add the `ClientAuthConfig` struct and update the module constructor:

```go
package app

import (
	"embed"
	"encoding/json"
	"html/template"
	"net/http"

	"github.com/JaimeStill/herald/pkg/module"
	"github.com/JaimeStill/herald/pkg/web"
)

// ClientAuthConfig holds the browser-safe subset of auth configuration
// needed by the web client for MSAL.js initialization.
type ClientAuthConfig struct {
	TenantID    string `json:"tenant_id"`
	ClientID    string `json:"client_id"`
	RedirectURI string `json:"redirect_uri"`
	Authority   string `json:"authority"`
}
```

Update `NewModule` to accept auth config:

```go
func NewModule(basePath string, authCfg *ClientAuthConfig) (*module.Module, error) {
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

	var data any
	if authCfg != nil {
		authCfg.RedirectURI = basePath + "/"
		b, err := json.Marshal(authCfg)
		if err != nil {
			return nil, err
		}
		data = template.JS(b)
	}

	router := buildRouter(ts, data)
	return module.New(basePath, router), nil
}
```

Update `buildRouter` to accept and pass data:

```go
func buildRouter(ts *web.TemplateSet, data any) http.Handler {
	r := web.NewRouter()

	r.Handle("GET /dist/", web.DistServer(distFS, "dist", "/dist/"))

	if data != nil {
		r.SetFallback(ts.PageHandlerWithData("app.html", views[0], data))
	} else {
		r.SetFallback(ts.PageHandler("app.html", views[0]))
	}

	return r
}
```

### Step 3: Update `app/server/layouts/app.html`

Add conditional config script in `<head>` and user-menu placeholder in `<header>`:

```html
<!doctype html>
<html lang="en">
  <head>
    <base href="{{ .BasePath }}/" />
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>{{ .Title }} - Herald</title>
    <link rel="stylesheet" href="dist/{{ .Bundle }}.css" />
    {{ if .Data }}<script id="herald-config" type="application/json">{{ .Data }}</script>{{ end }}
  </head>

  <body>
    <header class="app-header">
      <a href="" class="brand">Herald</a>
      <nav>
        <a href="prompts">Prompts</a>
      </nav>
      {{ if .Data }}<div id="user-menu"></div>{{ end }}
    </header>
    <main id="app-content">{{ block "content" . }}{{ end }}</main>

    <script type="module" src="dist/{{ .Bundle }}.js"></script>
  </body>
</html>
```

### Step 4: Update `cmd/server/modules.go`

Pass auth config subset from `cfg.Auth` to `app.NewModule`:

```go
import (
	"encoding/json"
	"net/http"

	"github.com/JaimeStill/herald/app"
	"github.com/JaimeStill/herald/internal/api"
	"github.com/JaimeStill/herald/internal/config"
	"github.com/JaimeStill/herald/internal/infrastructure"
	"github.com/JaimeStill/herald/pkg/auth"
	"github.com/JaimeStill/herald/pkg/module"
)
```

Update `NewModules`:

```go
func NewModules(infra *infrastructure.Infrastructure, cfg *config.Config) (*Modules, error) {
	apiModule, err := api.NewModule(cfg, infra)
	if err != nil {
		return nil, err
	}

	var authCfg *app.ClientAuthConfig
	if cfg.Auth.Mode == auth.ModeAzure {
		authCfg = &app.ClientAuthConfig{
			TenantID:  cfg.Auth.TenantID,
			ClientID:  cfg.Auth.ClientID,
			Authority: cfg.Auth.Authority,
		}
	}

	appModule, err := app.NewModule("/app", authCfg)
	if err != nil {
		return nil, err
	}

	return &Modules{
		API: apiModule,
		App: appModule,
	}, nil
}
```

## Validation Criteria

- [ ] `app.NewModule` accepts auth config (browser-safe subset only, no `ClientSecret`)
- [ ] `cmd/server/modules.go` passes auth config from `cfg.Auth` to app module
- [ ] `app.html` conditionally renders `<script id="herald-config" type="application/json">` with tenant_id, client_id, redirect_uri, authority when auth mode is azure
- [ ] When auth mode is `none`, no config script tag is rendered
- [ ] `<div id="user-menu"></div>` placeholder conditionally added to header when auth is enabled
- [ ] `go vet ./...` passes
