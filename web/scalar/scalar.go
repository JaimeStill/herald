package scalar

import (
	"embed"
	"html/template"
	"net/http"

	"github.com/JaimeStill/herald/pkg/module"
)

//go:embed index.html scalar.css scalar.js
var staticFS embed.FS

// NewModule creates a module that serves the Scalar API reference UI at basePath.
func NewModule(basePath string) *module.Module {
	router := buildRouter(basePath)
	return module.New(basePath, router)
}

func buildRouter(basePath string) http.Handler {
	mux := http.NewServeMux()

	tmpl := template.Must(template.ParseFS(staticFS, "index.html"))
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		tmpl.Execute(w, map[string]string{"BasePath": basePath})
	})

	mux.Handle("GET /", http.FileServer(http.FS(staticFS)))

	return mux
}
