// Package streaming provides interfaces for reading streamed responses from LLM providers.
package streaming

import (
	"context"
	"io"
)

// SSEMedia is the MIME type for Server-Sent Events.
const SSEMedia = "text/event-stream"

// StreamLine represents a single unit of data read from a stream.
type StreamLine struct {
	// Data contains the raw bytes of the streamed content.
	Data []byte
	// Done indicates whether the stream has completed.
	Done bool
	// Err holds any error encountered while reading the stream.
	Err error
}

// StreamReader reads a streaming response and emits StreamLine values
// on a channel until the stream is exhausted or the context is cancelled.
type StreamReader interface {
	ReadStream(
		ctx context.Context,
		reader io.Reader,
	) <-chan StreamLine
}
