// Package app provides the web application module with embedded templates and assets.
// It serves the Lit 3.x SPA shell via Go templates and delivers bundled static assets.
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

// NewModule creates the web app module configured for the given base path.
// All routes under the base path serve the SPA shell template, with /dist/
// serving embedded static assets (JS, CSS).
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
