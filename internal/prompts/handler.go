package prompts

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	"github.com/JaimeStill/herald/pkg/handlers"
	"github.com/JaimeStill/herald/pkg/pagination"
	"github.com/JaimeStill/herald/pkg/routes"
)

// Handler provides HTTP endpoints for prompt operations.
type Handler struct {
	sys        System
	logger     *slog.Logger
	pagination pagination.Config
}

// SearchRequest combines pagination and filter criteria for the search endpoint.
type SearchRequest struct {
	pagination.PageRequest
	Filters
}

// NewHandler creates a Handler with the given system, logger, and pagination config.
func NewHandler(
	sys System,
	logger *slog.Logger,
	pagination pagination.Config,
) *Handler {
	return &Handler{
		sys:        sys,
		logger:     logger.With("handler", "prompts"),
		pagination: pagination,
	}
}

// Routes returns the route group definition for prompt endpoints.
func (h *Handler) Routes() routes.Group {
	return routes.Group{
		Prefix: "/prompts",
		Routes: []routes.Route{
			{Method: "GET", Pattern: "", Handler: h.List},
			{Method: "GET", Pattern: "/stages", Handler: h.Stages},
			{Method: "GET", Pattern: "/{id}", Handler: h.Find},
			{Method: "POST", Pattern: "", Handler: h.Create},
			{Method: "PUT", Pattern: "/{id}", Handler: h.Update},
			{Method: "DELETE", Pattern: "/{id}", Handler: h.Delete},
			{Method: "POST", Pattern: "/search", Handler: h.Search},
			{Method: "POST", Pattern: "/{id}/activate", Handler: h.Activate},
			{Method: "POST", Pattern: "/{id}/deactivate", Handler: h.Deactivate},
		},
	}
}

// List returns a paginated list of prompts with optional query parameter filters.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	page := pagination.PageRequestFromQuery(r.URL.Query(), h.pagination)
	filters := FiltersFromQuery(r.URL.Query())

	result, err := h.sys.List(r.Context(), page, filters)
	if err != nil {
		handlers.RespondError(w, h.logger, http.StatusInternalServerError, err)
		return
	}

	handlers.RespondJSON(w, http.StatusOK, result)
}

// Stages returns the list of valid workflow stages.
func (h *Handler) Stages(w http.ResponseWriter, r *http.Request) {
	handlers.RespondJSON(w, http.StatusOK, Stages())
}

// Find returns a single prompt by its UUID path parameter.
func (h *Handler) Find(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, ErrNotFound)
		return
	}

	prompt, err := h.sys.Find(r.Context(), id)
	if err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(err), err)
		return
	}

	handlers.RespondJSON(w, http.StatusOK, prompt)
}

// Create processes a JSON body to create a new prompt override.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var cmd CreateCommand
	if err := json.NewDecoder(r.Body).Decode(&cmd); err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, err)
		return
	}

	prompt, err := h.sys.Create(r.Context(), cmd)
	if err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(err), err)
		return
	}

	handlers.RespondJSON(w, http.StatusCreated, prompt)
}

// Update processes a JSON body to update an existing prompt override.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, ErrNotFound)
		return
	}

	var cmd UpdateCommand
	if err := json.NewDecoder(r.Body).Decode(&cmd); err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, err)
		return
	}

	prompt, err := h.sys.Update(r.Context(), id, cmd)
	if err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(err), err)
		return
	}

	handlers.RespondJSON(w, http.StatusOK, prompt)
}

// Delete removes a prompt by its UUID path parameter.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, ErrNotFound)
		return
	}

	if err := h.sys.Delete(r.Context(), id); err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(err), err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Search accepts a JSON body with pagination and filter criteria and returns matching prompts.
func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	var req SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, err)
		return
	}

	req.PageRequest.Normalize(h.pagination)

	result, err := h.sys.List(r.Context(), req.PageRequest, req.Filters)
	if err != nil {
		handlers.RespondError(w, h.logger, http.StatusInternalServerError, err)
		return
	}

	handlers.RespondJSON(w, http.StatusOK, result)
}

// Activate sets a prompt as the active override for its stage,
// atomically deactivating any currently active prompt for the same stage.
func (h *Handler) Activate(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, ErrNotFound)
		return
	}

	prompt, err := h.sys.Activate(r.Context(), id)
	if err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(err), err)
		return
	}

	handlers.RespondJSON(w, http.StatusOK, prompt)
}

// Deactivate clears the active flag on a prompt, allowing the stage
// to fall back to hard-coded default instructions.
func (h *Handler) Deactivate(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, ErrNotFound)
		return
	}

	prompt, err := h.sys.Deactivate(r.Context(), id)
	if err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(err), err)
		return
	}

	handlers.RespondJSON(w, http.StatusOK, prompt)
}
