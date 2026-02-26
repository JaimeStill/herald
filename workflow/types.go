package workflow

import (
	"slices"
	"time"

	"github.com/google/uuid"
)

const (
	KeyDocumentID = "document_id"
	KeyTempDir    = "temp_dir"
	KeyFilename   = "filename"
	KeyPageCount  = "page_count"
	KeyClassState = "classification_state"
)

// Confidence represents a categorical assessment of classification certainty.
type Confidence string

// Confidence levels for classification results.
const (
	ConfidenceHigh   Confidence = "HIGH"
	ConfidenceMedium Confidence = "MEDIUM"
	ConfidenceLow    Confidence = "LOW"
)

// WorkflowResult is the final output from a classification workflow execution.
type WorkflowResult struct {
	DocumentID  uuid.UUID           `json:"document_id"`
	Filename    string              `json:"filename"`
	PageCount   int                 `json:"page_count"`
	State       ClassificationState `json:"state"`
	CompletedAt time.Time           `json:"completed_at"`
}

// ClassificationPage holds per-page data accumulated during classification.
// ImagePath references the rendered page image in a temp directory.
// Enhance signals that this page should be re-rendered with adjusted settings.
type ClassificationPage struct {
	PageNumber    int      `json:"page_number"`
	ImagePath     string   `json:"image_path"`
	MarkingsFound []string `json:"markings_found"`
	Rationale     string   `json:"rationale"`
	Enhance       bool     `json:"enhance"`
	Enhancements  string   `json:"enhancements"`
}

// ClassificationState holds the running document classification accumulated across pages.
type ClassificationState struct {
	Classification string               `json:"classification"`
	Confidence     Confidence           `json:"confidence"`
	Rationale      string               `json:"rationale"`
	Pages          []ClassificationPage `json:"pages"`
}

// NeedsEnhance reports whether any page is flagged for enhancement.
func (s *ClassificationState) NeedsEnhance() bool {
	return slices.ContainsFunc(s.Pages, func(p ClassificationPage) bool {
		return p.Enhance
	})
}

// EnhancePages returns the indices of pages flagged for enhancement.
func (s *ClassificationState) EnhancePages() []int {
	var indices []int
	for i, p := range s.Pages {
		if p.Enhance {
			indices = append(indices, i)
		}
	}
	return indices
}
