package openapi

import "net/http"

// Spec represents an OpenAPI 3.1 specification document.
type Spec struct {
	OpenAPI    string               `json:"openapi"`
	Info       *Info                `json:"info"`
	Servers    []*Server            `json:"servers,omitempty"`
	Paths      map[string]*PathItem `json:"paths"`
	Components *Components          `json:"components,omitempty"`
}

// NewSpec creates a Spec with the given title, version, and default components.
func NewSpec(title, version string) *Spec {
	return &Spec{
		OpenAPI: "3.1.0",
		Info: &Info{
			Title:   title,
			Version: version,
		},
		Components: NewComponents(),
		Paths:      make(map[string]*PathItem),
	}
}

// AddServer appends a server URL to the spec.
func (s *Spec) AddServer(url string) {
	s.Servers = append(s.Servers, &Server{URL: url})
}

// SetDescription sets the API description in the info object.
func (s *Spec) SetDescription(desc string) {
	s.Info.Description = desc
}

// ServeSpec returns a handler that serves pre-serialized JSON spec bytes.
func ServeSpec(specBytes []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write(specBytes)
	}
}
