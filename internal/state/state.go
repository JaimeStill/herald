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
//
// Confidence and UndeterminedMarkings are legibility signals emitted by the
// vision stages (classify, enhance) — the only stages that see the pixels.
// Confidence is the model's per-page assessment of how readable this page's
// markings are. UndeterminedMarkings names marking positions that are present
// but cannot be read; a non-empty list after enhancement means "present but
// unreadable" and forces this page to LOW during aggregation. Both feed
// ClassificationState.AggregateConfidence and are in-memory only (not persisted).
type ClassificationPage struct {
	PageNumber           int              `json:"page_number"`
	ImagePath            string           `json:"image_path"`
	MarkingsFound        []string         `json:"markings_found"`
	UndeterminedMarkings []string         `json:"undetermined_markings,omitempty"`
	Confidence           Confidence       `json:"confidence,omitempty"`
	Rationale            string           `json:"rationale"`
	Enhancements         *EnhanceSettings `json:"enhancements,omitempty"`
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

// rank orders confidence levels for conservative comparison: LOW < MEDIUM < HIGH.
// Unknown values rank lowest so an unexpected value never inflates the result.
func rank(c Confidence) int {
	switch c {
	case ConfidenceHigh:
		return 2
	case ConfidenceMedium:
		return 1
	default:
		return 0
	}
}

// AggregateConfidence derives the document-level confidence deterministically
// from the per-page legibility signals, replacing the text-only finalize guess.
//
// It is the most conservative (lowest) per-page Confidence among content-bearing
// pages — pages that found at least one marking or flagged one as undetermined.
// Blank, redacted, and cover pages carry no markings and are ignored, so they
// never lower the result. A content-bearing page with undetermined markings
// (present but unreadable, even after enhancement) is floored to LOW, and a
// content-bearing page that reports no Confidence at all defaults to LOW rather
// than silently inflating the document. A document with no content-bearing pages
// (nothing classified found) is a confident UNCLASSIFIED and stays HIGH.
//
// The result is always one of HIGH, MEDIUM, or LOW — never empty — which the
// persisted confidence column requires (NOT NULL, CHECK IN ('HIGH','MEDIUM','LOW')).
func (s *ClassificationState) AggregateConfidence() Confidence {
	result := ConfidenceHigh
	for _, p := range s.Pages {
		if len(p.MarkingsFound) == 0 && len(p.UndeterminedMarkings) == 0 {
			continue
		}
		pc := p.Confidence
		if pc == "" || len(p.UndeterminedMarkings) > 0 {
			pc = ConfidenceLow
		}
		if rank(pc) < rank(result) {
			result = pc
		}
	}
	return result
}
