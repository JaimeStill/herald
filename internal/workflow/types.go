package workflow

import (
	"time"

	"github.com/google/uuid"

	"github.com/JaimeStill/herald/internal/state"
)

// WorkflowResult is the final output from a classification workflow execution.
type WorkflowResult struct {
	DocumentID  uuid.UUID                 `json:"document_id"`
	Filename    string                    `json:"filename"`
	PageCount   int                       `json:"page_count"`
	State       state.ClassificationState `json:"state"`
	CompletedAt time.Time                 `json:"completed_at"`
}
