// Package state defines the shared types that Herald's format handlers and
// classification workflow both consume. It lives as a leaf package so higher
// layers (format, workflow) can import it without cycles.
package state

import "slices"

// State-bag keys used by the workflow to pass request-scoped values between nodes.
// Each key carries a specific Go type; consumers type-assert on retrieval.
const (
	// KeyDocumentID carries a uuid.UUID identifying the document under classification.
	KeyDocumentID = "document_id"
	// KeyTempDir carries the absolute path to the per-request scratch directory.
	// The workflow owns this directory's lifecycle (created by Execute,
	// removed via defer os.RemoveAll).
	KeyTempDir = "temp_dir"
	// KeyFilename carries the original uploaded filename.
	KeyFilename = "filename"
	// KeyPageCount carries the int count of pages produced during extraction.
	KeyPageCount = "page_count"
	// KeyClassState carries the ClassificationState value accumulated across
	// workflow nodes (init, classify, enhance, finalize).
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

// EnhanceSettings captures the semantic intent of a page-level enhancement
// pass — what kind of visual adjustment to apply — without encoding how any
// particular rasterizer consumes them. Pointer fields distinguish "not set"
// from "explicitly set to neutral"; only non-nil fields influence the render.
// Format handlers translate these fields into their own tool-specific
// arguments (e.g. ImageMagick's -brightness-contrast and -modulate operators).
type EnhanceSettings struct {
	Brightness *int `json:"brightness,omitempty"`
	Contrast   *int `json:"contrast,omitempty"`
	Saturation *int `json:"saturation,omitempty"`
}

// ClassificationPage holds per-page data accumulated during classification.
// ImagePath references the rendered page image in a temp directory.
// Enhance signals that this page should be re-rendered with adjusted settings.
type ClassificationPage struct {
	PageNumber    int              `json:"page_number"`
	ImagePath     string           `json:"image_path"`
	MarkingsFound []string         `json:"markings_found"`
	Rationale     string           `json:"rationale"`
	Enhancements  *EnhanceSettings `json:"enhancements,omitempty"`
}

// Enhance reports whether this page is flagged for enhancement.
func (p *ClassificationPage) Enhance() bool {
	return p.Enhancements != nil
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
		return p.Enhance()
	})
}

// EnhancePages returns the indices of pages flagged for enhancement.
func (s *ClassificationState) EnhancePages() []int {
	var indices []int
	for i, p := range s.Pages {
		if p.Enhance() {
			indices = append(indices, i)
		}
	}
	return indices
}
