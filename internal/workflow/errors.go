// Package workflow implements the classification workflow for Herald.
// It provides the 4-node state graph (init → classify → enhance? → finalize),
// prompt composition, response parsing, and the top-level Execute function.
package workflow

import "errors"

// Sentinel errors for workflow operations.
var (
	ErrDocumentNotFound = errors.New("document not found")
	ErrRenderFailed     = errors.New("failed to render page images")
	ErrClassifyFailed   = errors.New("classification failed")
	ErrEnhanceFailed    = errors.New("enhancement failed")
	ErrFinalizeFailed   = errors.New("finalize failed")
)
