# 62 — pkg/web/ Template and Static File Infrastructure

## Problem Context

The web app module (#64) needs Go-side infrastructure for template rendering, static file serving, and SPA routing. The `pkg/web/` package provides these capabilities, ported from the proven `~/code/agent-lab/pkg/web/` patterns.

## Architecture Approach

Direct port of three files from agent-lab with Herald's module path. The existing `pkg/web/doc.go` placeholder is removed. Herald's `pkg/routes` package already has the `Route` type needed by `static.go`.

## Implementation

### Step 1: Remove placeholder

Delete `pkg/web/doc.go`.

### Step 2: Create `pkg/web/views.go`

```go
package web

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
)

type ViewDef struct {
	Route    string
	Template string
	Title    string
	Bundle   string
}

type ViewData struct {
	Title    string
	Bundle   string
	BasePath string
	Data     any
}

type TemplateSet struct {
	views    map[string]*template.Template
	basePath string
}

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

func (ts *TemplateSet) Render(w http.ResponseWriter, layoutName, viewPath string, data ViewData) error {
	t, ok := ts.views[viewPath]
	if !ok {
		return fmt.Errorf("template not found: %s", viewPath)
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	return t.ExecuteTemplate(w, layoutName, data)
}
```

### Step 3: Create `pkg/web/static.go`

```go
package web

import (
	"bytes"
	"embed"
	"io/fs"
	"net/http"
	"time"

	"github.com/JaimeStill/herald/pkg/routes"
)

func DistServer(fsys embed.FS, subdir, urlPrefix string) http.HandlerFunc {
	sub, err := fs.Sub(fsys, subdir)
	if err != nil {
		panic("failed to create sub-filesystem: " + err.Error())
	}
	server := http.StripPrefix(urlPrefix, http.FileServer(http.FS(sub)))
	return func(w http.ResponseWriter, r *http.Request) {
		server.ServeHTTP(w, r)
	}
}

func PublicFile(fsys embed.FS, subdir, filename string) http.HandlerFunc {
	path := subdir + "/" + filename
	return func(w http.ResponseWriter, r *http.Request) {
		data, err := fsys.ReadFile(path)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		http.ServeContent(w, r, filename, time.Time{}, bytes.NewReader(data))
	}
}

func PublicFileRoutes(fsys embed.FS, subdir string, files ...string) []routes.Route {
	routeList := make([]routes.Route, len(files))
	for i, file := range files {
		routeList[i] = routes.Route{
			Method:  "GET",
			Pattern: "/" + file,
			Handler: PublicFile(fsys, subdir, file),
		}
	}
	return routeList
}

func ServeEmbeddedFile(data []byte, contentType string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", contentType)
		w.WriteHeader(http.StatusOK)
		w.Write(data)
	}
}
```

### Step 4: Create `pkg/web/router.go`

```go
package web

import "net/http"

type Router struct {
	mux      *http.ServeMux
	fallback http.HandlerFunc
}

func NewRouter() *Router {
	return &Router{mux: http.NewServeMux()}
}

func (r *Router) SetFallback(handler http.HandlerFunc) {
	r.fallback = handler
}

func (r *Router) Handle(pattern string, handler http.Handler) {
	r.mux.Handle(pattern, handler)
}

func (r *Router) HandleFunc(pattern string, handler http.HandlerFunc) {
	r.mux.HandleFunc(pattern, handler)
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	_, pattern := r.mux.Handler(req)
	if pattern == "" && r.fallback != nil {
		r.fallback.ServeHTTP(w, req)
		return
	}
	r.mux.ServeHTTP(w, req)
}
```

## Validation Criteria

- [ ] `pkg/web/doc.go` placeholder removed
- [ ] `pkg/web/views.go` compiles with `TemplateSet`, `ViewDef`, `ViewData`, `NewTemplateSet()`, `PageHandler()`, `ErrorHandler()`, `Render()`
- [ ] `pkg/web/static.go` compiles with `DistServer()`, `PublicFile()`, `PublicFileRoutes()`, `ServeEmbeddedFile()`
- [ ] `pkg/web/router.go` compiles with `Router`, `NewRouter()`, `SetFallback()`, `Handle()`, `HandleFunc()`, `ServeHTTP()`
- [ ] `go vet ./...` passes
- [ ] `go build ./...` compiles cleanly
