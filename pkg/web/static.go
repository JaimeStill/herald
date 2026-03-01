package web

import (
	"bytes"
	"embed"
	"io/fs"
	"net/http"
	"time"

	"github.com/JaimeStill/herald/pkg/routes"
)

// DistServer returns a handler that serves files from an embedded filesystem.
// It strips the URL prefix and serves from the specified subdirectory.
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

// PublicFile returns a handler that serves a single file from an embedded filesystem.
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

// PublicFileRoutes generates routes for serving multiple files at root-level URLs.
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

// ServeEmbeddedFile returns a handler that serves raw bytes with the specified content type.
func ServeEmbeddedFile(data []byte, contentType string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", contentType)
		w.WriteHeader(http.StatusOK)
		w.Write(data)
	}
}
