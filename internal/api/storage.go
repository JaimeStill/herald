package api

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path"
	"strconv"

	"github.com/JaimeStill/herald/pkg/handlers"
	"github.com/JaimeStill/herald/pkg/routes"
	"github.com/JaimeStill/herald/pkg/storage"
)

type storageHandler struct {
	store       storage.System
	logger      *slog.Logger
	maxListSize int32
}

func newStorageHandler(
	store storage.System,
	logger *slog.Logger,
	maxListSize int32,
) *storageHandler {
	return &storageHandler{
		store:       store,
		logger:      logger.With("handler", "storage"),
		maxListSize: maxListSize,
	}
}

func (h *storageHandler) routes() routes.Group {
	return routes.Group{
		Prefix: "/storage",
		Routes: []routes.Route{
			{Method: "GET", Pattern: "", Handler: h.list},
			{Method: "GET", Pattern: "/download/{key...}", Handler: h.download},
			{Method: "GET", Pattern: "/{key...}", Handler: h.find},
		},
	}
}

func (h *storageHandler) list(w http.ResponseWriter, r *http.Request) {
	prefix := r.URL.Query().Get("prefix")
	marker := r.URL.Query().Get("marker")

	maxResults, err := storage.ParseMaxResults(
		r.URL.Query().Get("max_results"),
		h.maxListSize,
	)
	if err != nil {
		handlers.RespondError(
			w, h.logger,
			http.StatusBadRequest, err,
		)
		return
	}

	result, err := h.store.List(
		r.Context(),
		prefix,
		marker,
		maxResults,
	)
	if err != nil {
		handlers.RespondError(
			w, h.logger,
			http.StatusInternalServerError, err,
		)
		return
	}

	handlers.RespondJSON(w, http.StatusOK, result)
}

func (h *storageHandler) find(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")

	meta, err := h.store.Find(r.Context(), key)
	if err != nil {
		handlers.RespondError(
			w, h.logger,
			storage.MapHTTPStatus(err), err,
		)
		return
	}

	handlers.RespondJSON(w, http.StatusOK, meta)
}

func (h *storageHandler) download(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")

	result, err := h.store.Download(r.Context(), key)
	if err != nil {
		handlers.RespondError(
			w, h.logger,
			storage.MapHTTPStatus(err), err,
		)
		return
	}
	defer result.Body.Close()

	w.Header().Set("Content-Type", result.ContentType)

	if result.ContentLength > 0 {
		w.Header().Set(
			"Content-Length",
			strconv.FormatInt(result.ContentLength, 10),
		)
	}
	w.Header().Set(
		"Content-Disposition",
		fmt.Sprintf("attachment; filename=%q", path.Base(key)),
	)
	w.WriteHeader(http.StatusOK)
	io.Copy(w, result.Body)
}
