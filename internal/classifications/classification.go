// Package classifications implements the classification domain for Herald.
// It provides types, data access, and business logic for storing, querying,
// validating, and updating classification results produced by the workflow engine.
package classifications

import (
	"time"

	"github.com/google/uuid"
)

// Classification represents a stored classification result for a document.
// It mirrors the classifications table schema with flattened workflow metadata.
type Classification struct {
	ID             uuid.UUID  `json:"id"`
	DocumentID     uuid.UUID  `json:"document_id"`
	Classification string     `json:"classification"`
	Confidence     string     `json:"confidence"`
	MarkingsFound  []string   `json:"markings_found"`
	Rationale      string     `json:"rationale"`
	ClassifiedAt   time.Time  `json:"classified_at"`
	ModelName      string     `json:"model_name"`
	ProviderName   string     `json:"provider_name"`
	ValidatedBy    *string    `json:"validated_by"`
	ValidatedAt    *time.Time `json:"validated_at"`
}

// ValidateCommand carries the data needed to validate a classification.
// ValidatedBy identifies the human who confirmed the AI classification.
type ValidateCommand struct {
	ValidatedBy string `json:"validated_by"`
}

// UpdateCommand carries the data needed to manually update a classification.
// Classification and Rationale overwrite the AI-produced values.
// UpdatedBy identifies the human who made the update (stored as validated_by).
type UpdateCommand struct {
	Classification string `json:"classification"`
	Rationale      string `json:"rationale"`
	UpdatedBy      string `json:"updated_by"`
}
