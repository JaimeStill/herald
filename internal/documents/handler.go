package documents

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/pdfcpu/pdfcpu/pkg/api"

	"github.com/JaimeStill/herald/pkg/handlers"
	"github.com/JaimeStill/herald/pkg/pagination"
	"github.com/JaimeStill/herald/pkg/routes"
)

// Handler provides HTTP endpoints for document operations.
type Handler struct {
	sys           System
	logger        *slog.Logger
	pagination    pagination.Config
	maxUploadSize int64
}

// SearchRequest combines pagination and filter criteria for the search endpoint.
type SearchRequest struct {
	pagination.PageRequest
	Filters
}

// NewHandler creates a Handler with the given system, logger, pagination config, and upload size limit.
func NewHandler(
	sys System,
	logger *slog.Logger,
	pagination pagination.Config,
	maxUploadSize int64,
) *Handler {
	return &Handler{
		sys:           sys,
		logger:        logger.With("handler", "documents"),
		pagination:    pagination,
		maxUploadSize: maxUploadSize,
	}
}

// Routes returns the route group definition for document endpoints.
func (h *Handler) Routes() routes.Group {
	return routes.Group{
		Prefix: "/documents",
		Routes: []routes.Route{
			{Method: "GET", Pattern: "", Handler: h.List},
			{Method: "GET", Pattern: "/{id}", Handler: h.Find},
			{Method: "POST", Pattern: "", Handler: h.Upload},
			{Method: "POST", Pattern: "/search", Handler: h.Search},
			{Method: "DELETE", Pattern: "/{id}", Handler: h.Delete},
		},
	}
}

// List returns a paginated list of documents with optional query parameter filters.
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

// Find returns a single document by its UUID path parameter.
func (h *Handler) Find(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, ErrInvalidFile)
		return
	}

	doc, err := h.sys.Find(r.Context(), id)
	if err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(err), err)
		return
	}

	handlers.RespondJSON(w, http.StatusOK, doc)
}

// Search accepts a JSON body with pagination and filter criteria and returns matching documents.
func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	var req SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, ErrInvalidFile)
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

// Upload processes a multipart form upload containing a file and external system metadata.
// Extracts PDF page count automatically for PDF files using pdfcpu.
func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(h.maxUploadSize); err != nil {
		handlers.RespondError(w, h.logger, http.StatusRequestEntityTooLarge, ErrFileTooLarge)
		return
	}

	externalID, err := strconv.Atoi(r.FormValue("external_id"))
	if err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, ErrInvalidFile)
		return
	}

	externalPlatform := r.FormValue("external_platform")
	if externalPlatform == "" {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, ErrInvalidFile)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, ErrInvalidFile)
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, ErrInvalidFile)
		return
	}

	contentType := detectContentType(header.Header.Get("Content-Type"), data)
	pageCount := extractPDFPageCount(h.logger, data, contentType)

	cmd := CreateCommand{
		Data:             data,
		Filename:         header.Filename,
		ContentType:      contentType,
		ExternalID:       externalID,
		ExternalPlatform: externalPlatform,
		PageCount:        pageCount,
	}

	doc, err := h.sys.Create(r.Context(), cmd)
	if err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(err), err)
		return
	}

	handlers.RespondJSON(w, http.StatusCreated, doc)
}

// Delete removes a document by its UUID path parameter.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, ErrInvalidFile)
		return
	}

	if err := h.sys.Delete(r.Context(), id); err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(err), err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func detectContentType(header string, data []byte) string {
	header = strings.TrimSpace(header)
	if header != "" && header != "application/octet-stream" {
		return header
	}
	return http.DetectContentType(data)
}

func extractPDFPageCount(logger *slog.Logger, data []byte, contentType string) *int {
	if contentType != "application/pdf" {
		return nil
	}

	count, err := api.PageCount(bytes.NewReader(data), nil)
	if err != nil {
		logger.Warn("failed to extract PDF page count", "error", err)
		return nil
	}

	return &count
}
