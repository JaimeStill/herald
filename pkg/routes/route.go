package routes

import (
	"net/http"

	"github.com/JaimeStill/herald/pkg/openapi"
)

// Route binds an HTTP method and pattern to a handler with optional OpenAPI metadata.
type Route struct {
	Method  string
	Pattern string
	Handler http.HandlerFunc
	OpenAPI *openapi.Operation
}
