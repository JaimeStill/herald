package classifications

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	"github.com/JaimeStill/herald/pkg/handlers"
	"github.com/JaimeStill/herald/pkg/pagination"
	"github.com/JaimeStill/herald/pkg/routes"
)

// Handler provides HTTP endpoints for classification operations.
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
		logger:     logger.With("handler", "classifications"),
		pagination: pagination,
	}
}

// Routes returns the route group definition for classification endpoints.
func (h *Handler) Routes() routes.Group {
	return routes.Group{
		Prefix: "/classifications",
		Routes: []routes.Route{
			{Method: "GET", Pattern: "", Handler: h.List},
			{Method: "GET", Pattern: "/{id}", Handler: h.Find},
			{Method: "GET", Pattern: "/document/{id}", Handler: h.FindByDocument},
			{Method: "POST", Pattern: "/search", Handler: h.Search},
			{Method: "POST", Pattern: "/{documentId}", Handler: h.Classify},
			{Method: "POST", Pattern: "/{id}/validate", Handler: h.Validate},
			{Method: "PUT", Pattern: "/{id}", Handler: h.Update},
			{Method: "DELETE", Pattern: "/{id}", Handler: h.Delete},
		},
	}
}

// List returns a paginated list of classifications with optional query parameter filters.
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

// Find returns a single classification by its UUID path parameter.
func (h *Handler) Find(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, ErrNotFound)
		return
	}

	c, err := h.sys.Find(r.Context(), id)
	if err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(err), err)
		return
	}

	handlers.RespondJSON(w, http.StatusOK, c)
}

// FindByDocument returns the classification associated with a document UUID path parameter.
func (h *Handler) FindByDocument(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, ErrNotFound)
		return
	}

	c, err := h.sys.FindByDocument(r.Context(), id)
	if err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(err), err)
		return
	}

	handlers.RespondJSON(w, http.StatusOK, c)
}

// Search accepts a JSON body with pagination and filter criteria and returns matching classifications.
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

// Classify executes the classification workflow for a document identified by the documentId path parameter.
// Returns 201 with the classification result on success.
func (h *Handler) Classify(w http.ResponseWriter, r *http.Request) {
	documentID, err := uuid.Parse(r.PathValue("documentId"))
	if err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, ErrNotFound)
		return
	}

	c, err := h.sys.Classify(r.Context(), documentID)
	if err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(err), err)
		return
	}

	handlers.RespondJSON(w, http.StatusCreated, c)
}

// Validate marks a classification as human-validated by decoding a ValidateCommand JSON body.
// Transitions the associated document status from review to complete.
func (h *Handler) Validate(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, ErrNotFound)
		return
	}

	var cmd ValidateCommand
	if err := json.NewDecoder(r.Body).Decode(&cmd); err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, err)
		return
	}

	c, err := h.sys.Validate(r.Context(), id, cmd)
	if err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(err), err)
		return
	}

	handlers.RespondJSON(w, http.StatusOK, c)
}

// Update manually overwrites a classification's result by decoding an UpdateCommand JSON body.
// Transitions the associated document status from review to complete.
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

	c, err := h.sys.Update(r.Context(), id, cmd)
	if err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(err), err)
		return
	}

	handlers.RespondJSON(w, http.StatusOK, c)
}

// Delete removes a classification by its UUID path parameter.
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
