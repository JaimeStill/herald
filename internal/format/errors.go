package format

import "errors"

// ErrUnsupportedFormat is returned by Registry.Lookup (and wrapped by
// higher layers such as documents.ErrUnsupportedContentType) when a content
// type has no registered handler. Use errors.Is to detect it.
var ErrUnsupportedFormat = errors.New("unsupported document format")
