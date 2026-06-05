package ollama

import (
	"context"
	"fmt"
	"maps"
	"net/http"
	"strings"

	"github.com/tailored-agentic-units/protocol"
	"github.com/tailored-agentic-units/protocol/config"
	protostreaming "github.com/tailored-agentic-units/protocol/streaming"
	"github.com/tailored-agentic-units/provider"
	"github.com/tailored-agentic-units/provider/streaming"
)

// Register registers the Ollama provider with the global provider registry.
func Register() {
	provider.Register("ollama", NewOllama)
}

// NewOllama creates a new OllamaProvider from configuration.
// Automatically adds /v1 suffix to base URL if not present for OpenAI compatibility.
// Supports optional authentication via "auth_type" and "token" options.
func NewOllama(c *config.ProviderConfig) (provider.Provider, error) {
	baseURL := c.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	if !strings.HasSuffix(baseURL, "/v1") {
		baseURL = strings.TrimSuffix(baseURL, "/") + "/v1"
	}

	return &OllamaProvider{
		BaseProvider: provider.NewBaseProvider(c.Name, baseURL),
		options:      c.Options,
		stream:       streaming.NewSSEReader(),
	}, nil
}

// OllamaProvider implements Provider for Ollama services with OpenAI-compatible API.
// Supports local and remote Ollama instances with optional authentication.
type OllamaProvider struct {
	*provider.BaseProvider
	options map[string]any
	stream  protostreaming.StreamReader
}

// Endpoint returns the full Ollama endpoint URL for a protocol.
// Supports chat, vision, tools (all use /chat/completions), and embeddings (/embeddings).
// Returns an error if the protocol is not supported.
func (p *OllamaProvider) Endpoint(proto protocol.Protocol) (string, error) {
	endpoints := map[protocol.Protocol]string{
		protocol.Chat:       "/chat/completions",
		protocol.Vision:     "/chat/completions",
		protocol.Tools:      "/chat/completions",
		protocol.Embeddings: "/embeddings",
	}

	endpoint, exists := endpoints[proto]
	if !exists {
		return "", fmt.Errorf("protocol %s not supported by Ollama", proto)
	}

	return fmt.Sprintf("%s%s", p.BaseURL(), endpoint), nil
}

// Stream returns the SSE reader for Ollama streaming responses.
func (p *OllamaProvider) Stream() protostreaming.StreamReader {
	return p.stream
}

// PrepareRequest prepares a standard (non-streaming) Ollama request.
// Returns an error if the endpoint is invalid.
func (p *OllamaProvider) PrepareRequest(ctx context.Context, proto protocol.Protocol, body []byte, headers map[string]string) (*provider.Request, error) {
	endpoint, err := p.Endpoint(proto)
	if err != nil {
		return nil, err
	}

	return &provider.Request{
		URL:     endpoint,
		Headers: headers,
		Body:    body,
	}, nil
}

// PrepareStreamRequest prepares a streaming Ollama request.
// Adds streaming-specific headers (Accept: text/event-stream, Cache-Control: no-cache).
// Returns an error if the endpoint is invalid.
func (p *OllamaProvider) PrepareStreamRequest(ctx context.Context, proto protocol.Protocol, body []byte, headers map[string]string) (*provider.Request, error) {
	endpoint, err := p.Endpoint(proto)
	if err != nil {
		return nil, err
	}

	streamHeaders := make(map[string]string)
	maps.Copy(streamHeaders, headers)
	streamHeaders["Accept"] = protostreaming.SSEMedia
	streamHeaders["Cache-Control"] = "no-cache"

	return &provider.Request{
		URL:     endpoint,
		Headers: streamHeaders,
		Body:    body,
	}, nil
}

// SetHeaders sets optional authentication headers based on provider options.
func (p *OllamaProvider) SetHeaders(ctx context.Context, req *http.Request) error {
	if authType, ok := p.options["auth_type"].(string); ok {
		if token, ok := p.options["token"].(string); ok && token != "" {
			switch authType {
			case "bearer":
				req.Header.Set("Authorization", "Bearer "+token)
			case "api_key":
				headerName := "X-API-Key"
				if head, ok := p.options["auth_header"].(string); ok && head != "" {
					headerName = head
				}
				req.Header.Set(headerName, token)
			}
		}
	}

	return nil
}
