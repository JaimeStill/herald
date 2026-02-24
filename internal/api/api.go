// Package api assembles the API module with all domain systems and route registration.
package api

import (
	"net/http"

	"github.com/JaimeStill/herald/internal/config"
	"github.com/JaimeStill/herald/internal/infrastructure"
	"github.com/JaimeStill/herald/pkg/middleware"
	"github.com/JaimeStill/herald/pkg/module"
)

// NewModule creates the API module with all domain handlers and middleware.
func NewModule(cfg *config.Config, infra *infrastructure.Infrastructure) (*module.Module, error) {
	runtime := NewRuntime(cfg, infra)
	domain := NewDomain(runtime)

	mux := http.NewServeMux()
	registerRoutes(mux, domain, cfg, runtime)

	m := module.New(cfg.API.BasePath, mux)
	m.Use(middleware.CORS(&cfg.API.CORS))
	m.Use(middleware.Logger(runtime.Infrastructure.Logger))

	return m, nil
}
