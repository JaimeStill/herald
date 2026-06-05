// Package streaming provides the SSE stream reader implementation for providers
// that use Server-Sent Events for streaming responses.
package streaming

import (
	"bufio"
	"context"
	"io"
	"strings"

	protostreaming "github.com/tailored-agentic-units/protocol/streaming"
)

// SSEReader reads Server-Sent Events streams, parsing "data:" prefixed lines.
// It implements protocol/streaming.StreamReader.
type SSEReader struct{}

// NewSSEReader returns a new SSEReader.
func NewSSEReader() *SSEReader {
	return &SSEReader{}
}

// ReadStream parses an SSE stream, emitting each data line as a StreamLine.
func (r *SSEReader) ReadStream(
	ctx context.Context,
	reader io.Reader,
) <-chan protostreaming.StreamLine {
	output := make(chan protostreaming.StreamLine)

	go func() {
		defer close(output)

		scanner := bufio.NewReader(reader)

		for {
			line, err := scanner.ReadString('\n')
			if err == io.EOF {
				return
			}
			if err != nil {
				select {
				case output <- protostreaming.StreamLine{Err: err}:
				case <-ctx.Done():
				}
				return
			}

			line = strings.TrimSpace(line)

			if line == "" {
				continue
			}

			data, ok := strings.CutPrefix(line, "data: ")
			if !ok {
				continue
			}

			if data == "[DONE]" {
				select {
				case output <- protostreaming.StreamLine{Done: true}:
				case <-ctx.Done():
				}
				return
			}

			select {
			case output <- protostreaming.StreamLine{Data: []byte(data)}:
			case <-ctx.Done():
				return
			}
		}
	}()

	return output
}
