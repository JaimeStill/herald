package format

import (
	"github.com/tailored-agentic-units/protocol"
	"github.com/tailored-agentic-units/protocol/response"
)

// Format defines the interface for marshaling requests and parsing responses
// in a provider-specific wire format such as OpenAI or AWS Converse.
type Format interface {
	// Name returns the identifier for this format (e.g., "openai", "converse").
	Name() string

	// Marshal serializes the given protocol-specific data into a JSON request body.
	Marshal(p protocol.Protocol, data any) ([]byte, error)

	// Parse deserializes a JSON response body into a protocol-specific response type.
	Parse(p protocol.Protocol, body []byte) (any, error)

	// ParseStreamChunk parses a single streaming event into a StreamingResponse.
	ParseStreamChunk(p protocol.Protocol, data []byte) (*response.StreamingResponse, error)
}
