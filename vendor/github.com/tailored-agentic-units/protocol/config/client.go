package config

import "time"

// ClientConfig defines the configuration for the HTTP client layer.
// It includes timeout settings, retry behavior, and connection pooling parameters.
// Duration fields are strings parsed via time.ParseDuration (e.g., "24s", "2m", "1h30m").
type ClientConfig struct {
	Timeout            string      `json:"timeout"`
	Retry              RetryConfig `json:"retry"`
	ConnectionPoolSize int         `json:"connection_pool_size"`
	ConnectionTimeout  string      `json:"connection_timeout"`
}

// RetryConfig configures retry behavior for failed requests.
// Implements exponential backoff with jitter for transient failures.
type RetryConfig struct {
	MaxRetries        int     `json:"max_retries"`
	InitialBackoff    string  `json:"initial_backoff"`
	MaxBackoff        string  `json:"max_backoff"`
	BackoffMultiplier float64 `json:"backoff_multiplier"`
	Jitter            bool    `json:"jitter"`
}

// TimeoutDuration returns Timeout as a time.Duration.
func (c *ClientConfig) TimeoutDuration() time.Duration {
	d, _ := time.ParseDuration(c.Timeout)
	return d
}

// ConnectionTimeoutDuration returns ConnectionTimeout as a time.Duration.
func (c *ClientConfig) ConnectionTimeoutDuration() time.Duration {
	d, _ := time.ParseDuration(c.ConnectionTimeout)
	return d
}

// InitialBackoffDuration returns InitialBackoff as a time.Duration.
func (r *RetryConfig) InitialBackoffDuration() time.Duration {
	d, _ := time.ParseDuration(r.InitialBackoff)
	return d
}

// MaxBackoffDuration returns MaxBackoff as a time.Duration.
func (r *RetryConfig) MaxBackoffDuration() time.Duration {
	d, _ := time.ParseDuration(r.MaxBackoff)
	return d
}

// DefaultClientConfig creates a ClientConfig with default values.
func DefaultClientConfig() *ClientConfig {
	return &ClientConfig{
		Timeout:            "2m",
		Retry:              DefaultRetryConfig(),
		ConnectionPoolSize: 10,
		ConnectionTimeout:  "30s",
	}
}

// DefaultRetryConfig creates a RetryConfig with default values.
// Retries up to 3 times with exponential backoff starting at 1s, capped at 30s.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:        3,
		InitialBackoff:    "1s",
		MaxBackoff:        "30s",
		BackoffMultiplier: 2.0,
		Jitter:            true,
	}
}

// Merge combines the source ClientConfig into this ClientConfig.
// Non-empty/non-zero values from source override the current values.
func (c *ClientConfig) Merge(source *ClientConfig) {
	if source.Timeout != "" {
		c.Timeout = source.Timeout
	}

	if source.Retry.MaxRetries > 0 {
		c.Retry.MaxRetries = source.Retry.MaxRetries
	}

	if source.Retry.InitialBackoff != "" {
		c.Retry.InitialBackoff = source.Retry.InitialBackoff
	}

	if source.Retry.MaxBackoff != "" {
		c.Retry.MaxBackoff = source.Retry.MaxBackoff
	}

	if source.Retry.BackoffMultiplier > 0 {
		c.Retry.BackoffMultiplier = source.Retry.BackoffMultiplier
	}

	c.Retry.Jitter = source.Retry.Jitter

	if source.ConnectionPoolSize > 0 {
		c.ConnectionPoolSize = source.ConnectionPoolSize
	}

	if source.ConnectionTimeout != "" {
		c.ConnectionTimeout = source.ConnectionTimeout
	}
}
