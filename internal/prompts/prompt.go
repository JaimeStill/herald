// Package prompts implements the prompt override domain for Herald.
// It provides types, data access, and HTTP handlers for managing
// named prompt instruction overrides per workflow stage.
package prompts

import "github.com/google/uuid"

// Prompt represents a named instruction override for a workflow stage.
type Prompt struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	Stage        Stage     `json:"stage"`
	Instructions string    `json:"instructions"`
	Description  *string   `json:"description"`
	Active       bool      `json:"active"`
}

// CreateCommand carries the data needed to create a new prompt override.
type CreateCommand struct {
	Name         string  `json:"name"`
	Stage        Stage   `json:"stage"`
	Instructions string  `json:"instructions"`
	Description  *string `json:"description"`
}

// UpdateCommand carries the data needed to update an existing prompt override.
type UpdateCommand struct {
	Name         string  `json:"name"`
	Stage        Stage   `json:"stage"`
	Instructions string  `json:"instructions"`
	Description  *string `json:"description"`
}
