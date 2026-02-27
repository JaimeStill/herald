package api

import (
	"github.com/JaimeStill/herald/internal/config"
	"github.com/JaimeStill/herald/internal/infrastructure"
	"github.com/JaimeStill/herald/pkg/pagination"
)

// Runtime extends Infrastructure with API-specific configuration.
type Runtime struct {
	*infrastructure.Infrastructure
	Pagination pagination.Config
}

// NewRuntime creates an API runtime with a module-scoped logger.
func NewRuntime(cfg *config.Config, infra *infrastructure.Infrastructure) *Runtime {
	return &Runtime{
		Infrastructure: &infrastructure.Infrastructure{
			Agent:     cfg.Agent,
			Lifecycle: infra.Lifecycle,
			Logger:    infra.Logger.With("module", "api"),
			Database:  infra.Database,
			Storage:   infra.Storage,
		},
		Pagination: cfg.API.Pagination,
	}
}
