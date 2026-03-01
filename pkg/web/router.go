package web

import "net/http"

// Router wraps http.ServeMux with optional fallback handling for unmatched routes.
// Use SetFallback to configure SPA catch-all behavior.
type Router struct {
	mux      *http.ServeMux
	fallback http.HandlerFunc
}

// NewRouter creates a Router with default ServeMux behavior.
func NewRouter() *Router {
	return &Router{mux: http.NewServeMux()}
}

// SetFallback configures the handler for unmatched routes.
func (r *Router) SetFallback(handler http.HandlerFunc) {
	r.fallback = handler
}

// Handle registers a handler for the given pattern.
func (r *Router) Handle(pattern string, handler http.Handler) {
	r.mux.Handle(pattern, handler)
}

// HandleFunc registers a handler function for the given pattern.
func (r *Router) HandleFunc(pattern string, handler http.HandlerFunc) {
	r.mux.HandleFunc(pattern, handler)
}

// ServeHTTP implements http.Handler with optional fallback for unmatched routes.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	_, pattern := r.mux.Handler(req)
	if pattern == "" && r.fallback != nil {
		r.fallback.ServeHTTP(w, req)
		return
	}
	r.mux.ServeHTTP(w, req)
}
