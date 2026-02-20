package storage

import "errors"

var (
	// ErrNotFound indicates the requested blob does not exist.
	ErrNotFound = errors.New("blob not found")
	// ErrEmptyKey indicates an empty storage key was provided.
	ErrEmptyKey = errors.New("storage key must not be empty")
	// ErrInvalidKey indicates the storage key contains a path traversal segment.
	ErrInvalidKey = errors.New("storage key contains invalid path segment")
)
