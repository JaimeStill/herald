// Package app provides the web application module with embedded templates and assets.
// It serves the Lit 3.x SPA shell via Go templates and delivers bundled static assets.
package app

import (
	"embed"
	"encoding/json"
	"html/template"
	"net/http"

	"github.com/JaimeStill/herald/pkg/auth"
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

// ClientAuthConfig holds the browser-safe subset of auth configuration
// injected into the HTML template for MSAL.js initialization. ClientSecret
// is deliberately excluded — only fields safe for client exposure are included.
type ClientAuthConfig struct {
	TenantID      string             `json:"tenant_id"`
	ClientID      string             `json:"client_id"`
	RedirectURI   string             `json:"redirect_uri"`
	Authority     string             `json:"authority"`
	CacheLocation auth.CacheLocation `json:"cache_location"`
}

// NewModule creates the web app module configured for the given base path.
// All routes under the base path serve the SPA shell template, with /dist/
// serving embedded static assets (JS, CSS).
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
