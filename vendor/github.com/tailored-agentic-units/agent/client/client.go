package client

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/tailored-agentic-units/agent/request"
	"github.com/tailored-agentic-units/format"
	"github.com/tailored-agentic-units/protocol/config"
	"github.com/tailored-agentic-units/protocol/response"
)

// Client provides the interface for executing LLM protocol requests.
// It orchestrates HTTP execution with retry logic and health tracking.
// Provider and model come from requests, enabling flexible request composition.
type Client interface {
	// HTTPClient returns a configured HTTP client.
	// Creates a new client on each call with timeout and connection pool settings.
	HTTPClient() *http.Client

	// Execute executes a protocol request and returns the parsed response.
	// Automatically retries on transient failures (HTTP 408/429 and 5xx, network errors).
	Execute(ctx context.Context, req request.Request) (any, error)

	// ExecuteStream executes a streaming protocol request and returns a channel of responses.
	// The channel is closed when streaming completes or context is cancelled.
	ExecuteStream(ctx context.Context, req request.Request) (<-chan *response.StreamingResponse, error)

	// IsHealthy returns the current health status of the client.
	// Set to false after request failures, true after successful requests.
	IsHealthy() bool
}

// client implements the Client interface with HTTP orchestration.
type client struct {
	config *config.ClientConfig

	mutex      sync.RWMutex
	healthy    bool
	lastHealth time.Time
}

// New creates a new Client from configuration.
// Initializes HTTP settings and health tracking.
func New(cfg *config.ClientConfig) Client {
	return &client{
		config:     cfg,
		healthy:    true,
		lastHealth: time.Now(),
	}
}

// HTTPClient creates and returns a configured HTTP client.
// Each call creates a new client with timeout and connection pool settings from configuration.
func (c *client) HTTPClient() *http.Client {
	return &http.Client{
		Timeout: c.config.TimeoutDuration(),
		Transport: &http.Transport{
			MaxIdleConns:        c.config.ConnectionPoolSize,
			MaxIdleConnsPerHost: c.config.ConnectionPoolSize,
			IdleConnTimeout:     c.config.ConnectionTimeoutDuration(),
		},
	}
}

// Execute executes a standard (non-streaming) protocol request.
// Executes with retry on transient failures.
func (c *client) Execute(ctx context.Context, req request.Request) (any, error) {
	return doWithRetry(ctx, c.config.Retry, func(ctx context.Context) (any, error) {
		return c.execute(ctx, req)
	})
}

// execute performs a single HTTP request attempt without retry logic.
// Returns HTTPStatusError for bad status codes, which retry logic evaluates.
func (c *client) execute(ctx context.Context, req request.Request) (any, error) {
	prov := req.Provider()
	proto := req.Protocol()

	// Marshal request body through format
	body, err := req.Marshal()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Prepare provider request
	providerRequest, err := prov.PrepareRequest(ctx, proto, body, req.Headers())
	if err != nil {
		return nil, fmt.Errorf("failed to prepare request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(
		ctx,
		"POST",
		providerRequest.URL,
		bytes.NewBuffer(providerRequest.Body),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	for key, value := range providerRequest.Headers {
		httpReq.Header.Set(key, value)
	}
	if err := prov.SetHeaders(ctx, httpReq); err != nil {
		return nil, fmt.Errorf("failed to set provider headers: %w", err)
	}

	// Execute HTTP request
	httpClient := c.HTTPClient()
	resp, err := httpClient.Do(httpReq)
	if err != nil {
		c.setHealthy(false)
		return nil, err
	}
	defer resp.Body.Close()

	// Check for non-OK status
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		c.setHealthy(false)
		return nil, &HTTPStatusError{
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
			Body:       bodyBytes,
		}
	}

	// Parse response through format
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		c.setHealthy(false)
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Get format from the request — requests carry a reference to the format
	// through their Marshal method. We need the format for parsing, so we
	// extract it from the request if it implements the formatAccessor interface.
	var f format.Format
	if fa, ok := req.(formatAccessor); ok {
		f = fa.Format()
	}

	if f == nil {
		c.setHealthy(false)
		return nil, fmt.Errorf("request does not provide a format for response parsing")
	}

	result, err := f.Parse(proto, bodyBytes)
	if err != nil {
		c.setHealthy(false)
		return nil, err
	}

	c.setHealthy(true)
	return result, nil
}

// formatAccessor is an optional interface that requests can implement
// to expose the format used for response parsing.
type formatAccessor interface {
	Format() format.Format
}

// ExecuteStream executes a streaming protocol request.
// Verifies protocol supports streaming and executes streaming flow.
func (c *client) ExecuteStream(ctx context.Context, req request.Request) (<-chan *response.StreamingResponse, error) {
	proto := req.Protocol()

	if !proto.SupportsStreaming() {
		return nil, fmt.Errorf("protocol %s does not support streaming", proto)
	}

	return c.executeStream(ctx, req)
}

// executeStream performs the streaming HTTP request.
// Streaming requests are not retried — they fail immediately on error.
func (c *client) executeStream(ctx context.Context, req request.Request) (<-chan *response.StreamingResponse, error) {
	prov := req.Provider()
	proto := req.Protocol()

	// Marshal request body through format
	body, err := req.Marshal()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Prepare streaming request
	providerRequest, err := prov.PrepareStreamRequest(ctx, proto, body, req.Headers())
	if err != nil {
		return nil, fmt.Errorf("failed to prepare streaming request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(
		ctx,
		"POST",
		providerRequest.URL,
		bytes.NewBuffer(providerRequest.Body),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	for key, value := range providerRequest.Headers {
		httpReq.Header.Set(key, value)
	}
	if err := prov.SetHeaders(ctx, httpReq); err != nil {
		return nil, fmt.Errorf("failed to set provider headers: %w", err)
	}

	// Execute HTTP request
	httpClient := c.HTTPClient()
	resp, err := httpClient.Do(httpReq)
	if err != nil {
		c.setHealthy(false)
		return nil, fmt.Errorf("streaming request failed: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		c.setHealthy(false)
		return nil, fmt.Errorf("streaming request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Get format and stream reader from the request/provider
	var f format.Format
	if fa, ok := req.(formatAccessor); ok {
		f = fa.Format()
	}

	if f == nil {
		resp.Body.Close()
		c.setHealthy(false)
		return nil, fmt.Errorf("request does not provide a format for stream parsing")
	}

	streamReader := prov.Stream()
	if streamReader == nil {
		resp.Body.Close()
		c.setHealthy(false)
		return nil, fmt.Errorf("provider does not support streaming")
	}

	// Read the stream and parse each chunk through format
	lines := streamReader.ReadStream(ctx, resp.Body)

	output := make(chan *response.StreamingResponse)
	go func() {
		defer close(output)
		defer resp.Body.Close()

		for line := range lines {
			if line.Err != nil {
				sr := &response.StreamingResponse{Error: line.Err}
				select {
				case output <- sr:
				case <-ctx.Done():
					return
				}
				continue
			}

			if line.Done {
				break
			}

			chunk, err := f.ParseStreamChunk(proto, line.Data)
			if err != nil {
				sr := &response.StreamingResponse{Error: err}
				select {
				case output <- sr:
				case <-ctx.Done():
					return
				}
				continue
			}

			if chunk == nil {
				continue
			}

			select {
			case output <- chunk:
			case <-ctx.Done():
				return
			}
		}
		c.setHealthy(true)
	}()

	return output, nil
}

// IsHealthy returns the current health status.
// Thread-safe for concurrent access via read mutex.
func (c *client) IsHealthy() bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.healthy
}

// setHealthy updates the health status with timestamp.
// Thread-safe via write mutex.
func (c *client) setHealthy(healthy bool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.healthy = healthy
	c.lastHealth = time.Now()
}
