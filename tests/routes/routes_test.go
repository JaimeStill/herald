package routes_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/JaimeStill/herald/pkg/openapi"
	"github.com/JaimeStill/herald/pkg/routes"
)

func TestRegisterHandlers(t *testing.T) {
	mux := http.NewServeMux()
	spec := openapi.NewSpec("Test", "1.0.0")

	routes.Register(mux, "/api", spec, routes.Group{
		Prefix: "/items",
		Tags:   []string{"Items"},
		Routes: []routes.Route{
			{
				Method:  "GET",
				Pattern: "",
				Handler: func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				},
				OpenAPI: &openapi.Operation{Summary: "List items"},
			},
			{
				Method:  "GET",
				Pattern: "/{id}",
				Handler: func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				},
				OpenAPI: &openapi.Operation{Summary: "Get item"},
			},
		},
	})

	tests := []struct {
		name    string
		method  string
		path    string
		wantOK  bool
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

func TestAddToSpecPopulatesPaths(t *testing.T) {
	spec := openapi.NewSpec("Test", "1.0.0")

	group := routes.Group{
		Prefix: "/docs",
		Tags:   []string{"Documents"},
		Routes: []routes.Route{
			{
				Method:  "GET",
				Pattern: "",
				Handler: nil,
				OpenAPI: &openapi.Operation{Summary: "List docs"},
			},
			{
				Method:  "POST",
				Pattern: "",
				Handler: nil,
				OpenAPI: &openapi.Operation{Summary: "Create doc"},
			},
		},
	}

	group.AddToSpec("/api", spec)

	pathItem, ok := spec.Paths["/api/docs"]
	if !ok {
		t.Fatal("path /api/docs not found in spec")
	}
	if pathItem.Get == nil {
		t.Error("GET operation missing")
	}
	if pathItem.Post == nil {
		t.Error("POST operation missing")
	}
	if pathItem.Get.Summary != "List docs" {
		t.Errorf("GET summary: got %s", pathItem.Get.Summary)
	}
}

func TestAddToSpecInheritsTags(t *testing.T) {
	spec := openapi.NewSpec("Test", "1.0.0")

	group := routes.Group{
		Prefix: "/docs",
		Tags:   []string{"Documents"},
		Routes: []routes.Route{
			{
				Method:  "GET",
				Pattern: "",
				Handler: nil,
				OpenAPI: &openapi.Operation{Summary: "List docs"},
			},
		},
	}

	group.AddToSpec("/api", spec)

	op := spec.Paths["/api/docs"].Get
	if len(op.Tags) != 1 || op.Tags[0] != "Documents" {
		t.Errorf("tags: got %v, want [Documents]", op.Tags)
	}
}

func TestAddToSpecSkipsNilOpenAPI(t *testing.T) {
	spec := openapi.NewSpec("Test", "1.0.0")

	group := routes.Group{
		Prefix: "/internal",
		Routes: []routes.Route{
			{
				Method:  "GET",
				Pattern: "/health",
				Handler: nil,
				OpenAPI: nil,
			},
		},
	}

	group.AddToSpec("", spec)

	if _, ok := spec.Paths["/internal/health"]; ok {
		t.Error("nil OpenAPI routes should not appear in spec")
	}
}

func TestAddToSpecNestedGroups(t *testing.T) {
	spec := openapi.NewSpec("Test", "1.0.0")

	group := routes.Group{
		Prefix: "/api",
		Children: []routes.Group{
			{
				Prefix: "/v1",
				Tags:   []string{"V1"},
				Routes: []routes.Route{
					{
						Method:  "GET",
						Pattern: "/items",
						Handler: nil,
						OpenAPI: &openapi.Operation{Summary: "V1 items"},
					},
				},
			},
		},
	}

	group.AddToSpec("", spec)

	if _, ok := spec.Paths["/api/v1/items"]; !ok {
		t.Error("nested path /api/v1/items not found")
	}
}

func TestAddToSpecCopiesSchemas(t *testing.T) {
	spec := openapi.NewSpec("Test", "1.0.0")

	group := routes.Group{
		Prefix: "/docs",
		Schemas: map[string]*openapi.Schema{
			"Document": {Type: "object"},
		},
		Routes: []routes.Route{},
	}

	group.AddToSpec("/api", spec)

	if _, ok := spec.Components.Schemas["Document"]; !ok {
		t.Error("Document schema not copied to spec")
	}
}
