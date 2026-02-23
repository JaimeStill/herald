package api

import (
	"net/http"

	"github.com/JaimeStill/herald/internal/config"
	"github.com/JaimeStill/herald/pkg/routes"
)

func registerRoutes(
	mux *http.ServeMux,
	domain *Domain,
	cfg *config.Config,
) {
	routes.Register(
		mux,
		domain.Documents.Handler(cfg.API.MaxUploadSizeBytes()).Routes(),
	)
}
