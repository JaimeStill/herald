package routes_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/JaimeStill/herald/pkg/routes"
)

func TestRegisterHandlers(t *testing.T) {
	mux := http.NewServeMux()

	routes.Register(mux, routes.Group{
		Prefix: "/items",
		Routes: []routes.Route{
			{
				Method:  "GET",
				Pattern: "",
				Handler: func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				},
			},
			{
				Method:  "GET",
				Pattern: "/{id}",
				Handler: func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				},
			},
		},
	})

	tests := []struct {
		name   string
		method string
		path   string
		wantOK bool
	}{
		{"list items", "GET", "/items", true},
		{"get item", "GET", "/items/123", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(tt.method, tt.path, nil)
			mux.ServeHTTP(rec, req)

			if tt.wantOK && rec.Code != http.StatusOK {
				t.Errorf("status: got %d, want 200", rec.Code)
			}
		})
	}
}

func TestNestedGroups(t *testing.T) {
	mux := http.NewServeMux()

	routes.Register(mux, routes.Group{
		Prefix: "/api",
		Children: []routes.Group{
			{
				Prefix: "/v1",
				Routes: []routes.Route{
					{
						Method:  "GET",
						Pattern: "/items",
						Handler: func(w http.ResponseWriter, r *http.Request) {
							w.WriteHeader(http.StatusOK)
						},
					},
				},
			},
		},
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/items", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("nested route: got %d, want 200", rec.Code)
	}
}
