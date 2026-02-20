package routes

import (
	"maps"
	"net/http"

	"github.com/JaimeStill/herald/pkg/openapi"
)

// Group organizes routes under a common prefix with shared tags and schemas.
type Group struct {
	Prefix      string
	Tags        []string
	Description string
	Routes      []Route
	Children    []Group
	Schemas     map[string]*openapi.Schema
}

// AddToSpec populates the OpenAPI spec with operations from this group and its children.
func (g *Group) AddToSpec(basePath string, spec *openapi.Spec) {
	g.addOperations(basePath, spec)
}

func (g *Group) addOperations(parentPrefix string, spec *openapi.Spec) {
	fullPrefix := parentPrefix + g.Prefix

	maps.Copy(spec.Components.Schemas, g.Schemas)

	for _, route := range g.Routes {
		if route.OpenAPI == nil {
			continue
		}

		path := fullPrefix + route.Pattern
		op := route.OpenAPI

		if len(op.Tags) == 0 {
			op.Tags = g.Tags
		}

		if spec.Paths[path] == nil {
			spec.Paths[path] = &openapi.PathItem{}
		}

		switch route.Method {
		case "GET":
			spec.Paths[path].Get = op
		case "POST":
			spec.Paths[path].Post = op
		case "PUT":
			spec.Paths[path].Put = op
		case "DELETE":
			spec.Paths[path].Delete = op
		}
	}

	for _, child := range g.Children {
		child.addOperations(fullPrefix, spec)
	}
}

// Register adds all routes from the given groups to the mux and populates the OpenAPI spec.
func Register(mux *http.ServeMux, basePath string, spec *openapi.Spec, groups ...Group) {
	for _, group := range groups {
		group.AddToSpec(basePath, spec)
		registerGroup(mux, "", group)
	}
}

func registerGroup(mux *http.ServeMux, parentPrefix string, group Group) {
	fullPrefix := parentPrefix + group.Prefix
	for _, route := range group.Routes {
		pattern := route.Method + " " + fullPrefix + route.Pattern
		mux.HandleFunc(pattern, route.Handler)
	}
	for _, child := range group.Children {
		registerGroup(mux, fullPrefix, child)
	}
}
