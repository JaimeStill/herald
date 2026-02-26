package classifications

import (
	"errors"
	"net/http"
)

// Domain errors for classification operations.
var (
	ErrNotFound      = errors.New("classification not found")
	ErrDuplicate     = errors.New("classification already exists")
	ErrInvalidStatus = errors.New("document is not in review status")
)

// MapHTTPStatus maps classification domain errors to appropriate HTTP status codes.
func MapHTTPStatus(err error) int {
	if errors.Is(err, ErrNotFound) {
		return http.StatusNotFound
	}
	if errors.Is(err, ErrDuplicate) {
		return http.StatusConflict
	}
	if errors.Is(err, ErrInvalidStatus) {
		return http.StatusConflict
	}
	return http.StatusInternalServerError
}
