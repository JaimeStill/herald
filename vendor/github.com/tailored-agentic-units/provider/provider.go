package provider

import (
	"context"
	"net/http"

	"github.com/tailored-agentic-units/protocol"
	protostreaming "github.com/tailored-agentic-units/protocol/streaming"
)

// Provider defines the interface for LLM service provider implementations.
// Providers handle endpoint routing, authentication, request marshaling,
// and response processing for their specific service.
type Provider interface {
	// Name returns the provider identifier.
	Name() string

	// BaseURL returns the provider's base URL.
	BaseURL() string

	// Endpoint returns the full endpoint URL for a protocol.
	// Returns an error if the protocol is not supported by this provider.
	Endpoint(p protocol.Protocol) (string, error)

	// Stream returns the stream reader used for streaming responses.
	Stream() protostreaming.StreamReader

	// SetHeaders sets provider-specific authentication and custom headers on an HTTP request.
	// This is called after the request is created but before it is executed.
	SetHeaders(ctx context.Context, req *http.Request) error

	// PrepareRequest creates a Request for standard (non-streaming) protocol execution.
	// Accepts pre-marshaled request body and headers from the request structure.
	PrepareRequest(ctx context.Context, p protocol.Protocol, body []byte, headers map[string]string) (*Request, error)

	// PrepareStreamRequest creates a Request for streaming protocol execution.
	// Accepts pre-marshaled request body and headers, adds streaming-specific headers.
	PrepareStreamRequest(ctx context.Context, p protocol.Protocol, body []byte, headers map[string]string) (*Request, error)
}

// Request represents a prepared provider request with all necessary components for HTTP execution.
// This structure decouples request preparation from HTTP client execution.
type Request struct {
	// URL is the complete endpoint URL including query parameters.
	URL string

	// Headers contains protocol-specific and provider-specific headers.
	Headers map[string]string

	// Body is the marshaled request body ready for HTTP transmission.
	Body []byte
}
