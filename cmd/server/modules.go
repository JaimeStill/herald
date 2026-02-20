package main

import (
	"encoding/json"
	"net/http"

	"github.com/JaimeStill/herald/internal/api"
	"github.com/JaimeStill/herald/internal/config"
	"github.com/JaimeStill/herald/internal/infrastructure"
	"github.com/JaimeStill/herald/pkg/middleware"
	"github.com/JaimeStill/herald/pkg/module"
	"github.com/JaimeStill/herald/web/scalar"
)

type Modules struct {
	API    *module.Module
	Scalar *module.Module
}

func NewModules(infra *infrastructure.Infrastructure, cfg *config.Config) (*Modules, error) {
	apiModule, err := api.NewModule(cfg, infra)
	if err != nil {
		return nil, err
	}

	scalarModule := scalar.NewModule("/scalar")
	scalarModule.Use(middleware.Logger(infra.Logger))

	return &Modules{
		API:    apiModule,
		Scalar: scalarModule,
	}, nil
}

func (m *Modules) Mount(router *module.Router) {
	router.Mount(m.API)
	router.Mount(m.Scalar)
}

func buildRouter(infra *infrastructure.Infrastructure) *module.Router {
	router := module.NewRouter()

	router.HandleNative("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	router.HandleNative("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if !infra.Lifecycle.Ready() {
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]string{"status": "not ready"})
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
	})

	return router
}
