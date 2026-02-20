package api

import (
	"net/http"

	"github.com/JaimeStill/herald/internal/config"
	"github.com/JaimeStill/herald/pkg/openapi"
)

func registerRoutes(
	mux *http.ServeMux,
	spec *openapi.Spec,
	domain *Domain,
	cfg *config.Config,
) {
}
