// Package web provides infrastructure for serving web pages with Go templates,
// embedded static assets, and SPA-compatible routing.
package web

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
)

// ViewDef defines a page with its route, template file, title, and bundle name.
type ViewDef struct {
	Route    string
	Template string
	Title    string
	Bundle   string
}

// ViewData contains the data passed to page templates during rendering.
// BasePath enables portable URL generation in templates via {{ .BasePath }}.
type ViewData struct {
	Title    string
	Bundle   string
	BasePath string
	Data     any
}

// TemplateSet holds pre-parsed templates and a base path for URL generation.
// Templates are parsed once at startup, avoiding per-request overhead.
type TemplateSet struct {
	views    map[string]*template.Template
	basePath string
}

// NewTemplateSet creates a TemplateSet by parsing layout templates and cloning
// them for each view. The basePath is automatically included in ViewData for all
// handlers. Pre-parsing at startup enables fail-fast behavior and eliminates
// per-request template parsing overhead.
func NewTemplateSet(layoutFS, viewFS embed.FS, layoutGlob, viewSubdir, basePath string, views []ViewDef) (*TemplateSet, error) {
	layouts, err := template.ParseFS(layoutFS, layoutGlob)
	if err != nil {
		return nil, err
	}

	viewSub, err := fs.Sub(viewFS, viewSubdir)
	if err != nil {
		return nil, err
	}

	viewTemplates := make(map[string]*template.Template, len(views))
	for _, p := range views {
		t, err := layouts.Clone()
		if err != nil {
			return nil, fmt.Errorf("clone layouts for %s: %w", p.Template, err)
		}
		_, err = t.ParseFS(viewSub, p.Template)
		if err != nil {
			return nil, fmt.Errorf("parse template: %s: %w", p.Template, err)
		}
		viewTemplates[p.Template] = t
	}

	return &TemplateSet{
		views:    viewTemplates,
		basePath: basePath,
	}, nil
}

// ErrorHandler returns an HTTP handler that renders an error page with the given status code.
func (ts *TemplateSet) ErrorHandler(layout string, view ViewDef, status int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		data := ViewData{
			Title:    view.Title,
			Bundle:   view.Bundle,
			BasePath: ts.basePath,
		}
		if err := ts.Render(w, layout, view.Template, data); err != nil {
			http.Error(w, http.StatusText(status), status)
		}
	}
}

// PageHandler returns an HTTP handler that renders the given view.
func (ts *TemplateSet) PageHandler(layout string, view ViewDef) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data := ViewData{
			Title:    view.Title,
			Bundle:   view.Bundle,
			BasePath: ts.basePath,
		}
		if err := ts.Render(w, layout, view.Template, data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

// Render executes the named layout template with the given view data.
func (ts *TemplateSet) Render(w http.ResponseWriter, layoutName, viewPath string, data ViewData) error {
	t, ok := ts.views[viewPath]
	if !ok {
		return fmt.Errorf("template not found: %s", viewPath)
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	return t.ExecuteTemplate(w, layoutName, data)
}
