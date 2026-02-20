// Package api assembles the API module with all domain systems and route registration.
package api

import (
	"net/http"

	"github.com/JaimeStill/herald/internal/config"
	"github.com/JaimeStill/herald/internal/infrastructure"
	"github.com/JaimeStill/herald/pkg/middleware"
	"github.com/JaimeStill/herald/pkg/module"
	"github.com/JaimeStill/herald/pkg/openapi"
)

// NewModule creates the API module with all domain handlers and middleware.
func NewModule(cfg *config.Config, infra *infrastructure.Infrastructure) (*module.Module, error) {
	runtime := NewRuntime(cfg, infra)
	domain := NewDomain(runtime)

	spec := openapi.NewSpec(cfg.API.OpenAPI.Title, cfg.Version)
	spec.SetDescription(cfg.API.OpenAPI.Description)

	mux := http.NewServeMux()
	registerRoutes(mux, spec, domain, cfg)

	specBytes, err := openapi.MarshalJSON(spec)
	if err != nil {
		return nil, err
	}
	mux.HandleFunc("GET /openapi.json", openapi.ServeSpec(specBytes))

	m := module.New(cfg.API.BasePath, mux)
	m.Use(middleware.CORS(&cfg.API.CORS))
	m.Use(middleware.Logger(runtime.Infrastructure.Logger))

	return m, nil
}
