package prompts

import (
	"errors"
	"net/http"
)

// Domain errors for prompt operations.
var (
	ErrNotFound     = errors.New("prompt not found")
	ErrDuplicate    = errors.New("prompt name already exists")
	ErrInvalidStage = errors.New("stage must be init, classify, or enhance")
)

// MapHTTPStatus maps prompt domain errors to appropriate HTTP status codes.
func MapHTTPStatus(err error) int {
	if errors.Is(err, ErrNotFound) {
		return http.StatusNotFound
	}
	if errors.Is(err, ErrDuplicate) {
		return http.StatusConflict
	}
	if errors.Is(err, ErrInvalidStage) {
		return http.StatusBadRequest
	}
	return http.StatusInternalServerError
}
