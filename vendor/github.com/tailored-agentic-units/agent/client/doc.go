// Package client provides HTTP client orchestration for LLM protocol execution.
// It integrates providers, formats, and models to execute protocol requests
// with proper option management, HTTP configuration, retry logic, and health tracking.
//
// # Client Interface
//
// The Client interface orchestrates protocol execution:
//
//	type Client interface {
//	    HTTPClient() *http.Client
//	    Execute(ctx, req) (any, error)
//	    ExecuteStream(ctx, req) (<-chan *response.StreamingResponse, error)
//	    IsHealthy() bool
//	}
//
// # Creating a Client
//
// Clients are created from client configuration:
//
//	cfg := &config.ClientConfig{
//	    Timeout:            "2m",
//	    ConnectionTimeout:  "30s",
//	    ConnectionPoolSize: 10,
//	    Retry: config.RetryConfig{
//	        MaxRetries:     3,
//	        InitialBackoff: "1s",
//	        MaxBackoff:     "30s",
//	        Jitter:         true,
//	    },
//	}
//
//	c := client.New(cfg)
//
// # Retry Logic
//
// The client retries on transient failures:
//   - HTTP 408 (request timeout), 429 (rate limit), and all 5xx server errors
//   - Network operation errors (connection failures, timeouts)
//   - Temporary DNS errors
//
// Uses exponential backoff with optional jitter to prevent thundering herd.
//
// # Health Tracking
//
// The client tracks health status based on request success/failure.
// Thread-safe for concurrent health checks.
package client
