package azure

import (
	"context"
	"fmt"
	"maps"
	"net/http"

	"github.com/tailored-agentic-units/protocol"
	"github.com/tailored-agentic-units/protocol/config"
	protostreaming "github.com/tailored-agentic-units/protocol/streaming"
	"github.com/tailored-agentic-units/provider"
	"github.com/tailored-agentic-units/provider/streaming"
)

// Register registers the Azure provider with the global provider registry.
func Register() {
	provider.Register("azure", NewAzure)
}

// NewAzure creates a new AzureProvider from configuration.
// Requires "deployment", "auth_type", and "api_version" in options.
// For "api_key" and "bearer" auth types, "token" is also required.
// For "managed_identity", optional "resource" (token scope) and "client_id"
// (user-assigned identity) are supported.
// Returns an error if any required option is missing or auth_type is unsupported.
func NewAzure(c *config.ProviderConfig) (provider.Provider, error) {
	deployment, ok := c.Options["deployment"].(string)
	if !ok || deployment == "" {
		return nil, fmt.Errorf("deployment is required for Azure provider")
	}

	authType, ok := c.Options["auth_type"].(string)
	if !ok || authType == "" {
		return nil, fmt.Errorf("auth_type is required for Azure provider")
	}

	apiVersion, ok := c.Options["api_version"].(string)
	if !ok || apiVersion == "" {
		return nil, fmt.Errorf("api_version is required for Azure provider")
	}

	p := &AzureProvider{
		BaseProvider: provider.NewBaseProvider(c.Name, c.BaseURL),
		deployment:   deployment,
		authType:     authType,
		apiVersion:   apiVersion,
		stream:       streaming.NewSSEReader(),
	}

	if err := p.initAuth(c); err != nil {
		return nil, err
	}

	return p, nil
}

// AzureProvider implements Provider for Azure OpenAI Service.
// Supports deployment-based routing with API key, Entra ID (bearer token),
// and managed identity authentication.
type AzureProvider struct {
	*provider.BaseProvider
	deployment  string
	authType    string
	token       string
	apiVersion  string
	tokenSource *AzureTokenSource
	stream      protostreaming.StreamReader
}

// Endpoint returns the full Azure OpenAI endpoint URL for a protocol.
// Includes deployment name in path and api-version as query parameter.
// Supports chat, vision, tools (all use /deployments/{deployment}/chat/completions),
// and embeddings (/deployments/{deployment}/embeddings).
// Returns an error if the protocol is not supported.
func (p *AzureProvider) Endpoint(proto protocol.Protocol) (string, error) {
	basePath := fmt.Sprintf("/deployments/%s", p.deployment)

	endpoints := map[protocol.Protocol]string{
		protocol.Chat:       basePath + "/chat/completions",
		protocol.Vision:     basePath + "/chat/completions",
		protocol.Tools:      basePath + "/chat/completions",
		protocol.Embeddings: basePath + "/embeddings",
	}

	endpoint, exists := endpoints[proto]
	if !exists {
		return "", fmt.Errorf("protocol %s not supported by Azure", proto)
	}

	return fmt.Sprintf("%s%s?api-version=%s", p.BaseURL(), endpoint, p.apiVersion), nil
}

// Stream returns the SSE reader for Azure streaming responses.
func (p *AzureProvider) Stream() protostreaming.StreamReader {
	return p.stream
}

// PrepareRequest prepares a standard (non-streaming) Azure request.
// Returns an error if the endpoint is invalid.
func (p *AzureProvider) PrepareRequest(ctx context.Context, proto protocol.Protocol, body []byte, headers map[string]string) (*provider.Request, error) {
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

// PrepareStreamRequest prepares a streaming Azure request.
// Adds streaming-specific headers (Accept: text/event-stream, Cache-Control: no-cache).
// Returns an error if the endpoint is invalid.
func (p *AzureProvider) PrepareStreamRequest(ctx context.Context, proto protocol.Protocol, body []byte, headers map[string]string) (*provider.Request, error) {
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

// SetHeaders sets authentication headers on the HTTP request.
// Supports "api_key" (api-key header), "bearer" (Authorization: Bearer),
// and "managed_identity" (dynamic token acquisition via Azure identity).
func (p *AzureProvider) SetHeaders(ctx context.Context, req *http.Request) error {
	switch p.authType {
	case "api_key":
		if p.token != "" {
			req.Header.Set("api-key", p.token)
		}
	case "bearer":
		if p.token != "" {
			req.Header.Set("Authorization", "Bearer "+p.token)
		}
	case "managed_identity":
		token, err := p.tokenSource.GetToken(ctx)
		if err != nil {
			return fmt.Errorf("acquire managed identity token: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+token)
	}

	return nil
}

// initAuth initializes authentication based on the configured auth_type.
func (p *AzureProvider) initAuth(c *config.ProviderConfig) error {
	switch p.authType {
	case "api_key", "bearer":
		token, ok := c.Options["token"].(string)
		if !ok || token == "" {
			return fmt.Errorf("token is required for Azure provider with auth_type %q", p.authType)
		}
		p.token = token
	case "managed_identity":
		resource, _ := c.Options["resource"].(string)
		clientID, _ := c.Options["client_id"].(string)

		tokenSource, err := NewAzureTokenSource(resource, clientID)
		if err != nil {
			return fmt.Errorf("initialize managed identity: %w", err)
		}
		p.tokenSource = tokenSource
	default:
		return fmt.Errorf("unsupported auth_type %q for Azure provider", p.authType)
	}

	return nil
}
