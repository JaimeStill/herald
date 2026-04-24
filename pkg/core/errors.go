package core

import "errors"

// ErrParseFailed is returned when content cannot be parsed as JSON,
// either directly or from a markdown code fence.
var ErrParseFailed = errors.New("failed to parse response")
