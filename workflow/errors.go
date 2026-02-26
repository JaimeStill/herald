// Package workflow implements the classification workflow for Herald.
// It provides foundational types, prompt composition, and response parsing
// used by the 3-node state graph (init → classify → enhance?).
package workflow

import "errors"

// Sentinel errors for workflow operations.
var (
	ErrDocumentNotFound = errors.New("document not found")
	ErrRenderFailed     = errors.New("failed to render page images")
	ErrClassifyFailed   = errors.New("classification failed")
	ErrEnhanceFailed    = errors.New("enhancement failed")
)
