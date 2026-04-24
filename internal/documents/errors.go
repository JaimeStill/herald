package documents

import (
	"errors"
	"net/http"
)

// Domain errors for document operations.
var (
	ErrNotFound               = errors.New("document not found")
	ErrDuplicate              = errors.New("document already exists")
	ErrFileTooLarge           = errors.New("file exceeds maximum upload size")
	ErrInvalidFile            = errors.New("invalid file")
	ErrUnsupportedContentType = errors.New("unsupported content type")
)

// MapHTTPStatus maps document domain errors to appropriate HTTP status codes.
func MapHTTPStatus(err error) int {
	switch {
	case errors.Is(err, ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, ErrDuplicate):
		return http.StatusConflict
	case errors.Is(err, ErrFileTooLarge):
		return http.StatusRequestEntityTooLarge
	case
		errors.Is(err, ErrInvalidFile),
		errors.Is(err, ErrUnsupportedContentType):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}
