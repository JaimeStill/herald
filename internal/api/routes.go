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
	runtime *Runtime,
) {
	classificationsRoutes := domain.
		Classifications.
		Handler().
		Routes()

	documentsRoutes := domain.
		Documents.
		Handler(cfg.API.MaxUploadSizeBytes()).
		Routes()

	promptsRoutes := domain.
		Prompts.
		Handler().
		Routes()

	storageRoutes := newStorageHandler(
		runtime.Storage,
		runtime.Logger,
		cfg.Storage.MaxListSize,
	).routes()

	routes.Register(
		mux,
		classificationsRoutes,
		documentsRoutes,
		promptsRoutes,
		storageRoutes,
	)
}
