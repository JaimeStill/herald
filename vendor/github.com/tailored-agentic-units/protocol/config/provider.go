package config

import "maps"

// ProviderConfig defines the configuration for an LLM provider.
// It includes the provider name, base URL, and provider-specific options
// (e.g., deployment, API version, authentication type).
type ProviderConfig struct {
	Name    string         `json:"name"`
	BaseURL string         `json:"base_url"`
	Options map[string]any `json:"options"`
}

// DefaultProviderConfig creates a ProviderConfig with Ollama defaults.
// BaseURL is intentionally empty — providers auto-construct their default
// base URL when not explicitly configured (e.g., Ollama defaults to
// localhost:11434, Bedrock constructs from region).
func DefaultProviderConfig() *ProviderConfig {
	return &ProviderConfig{
		Name:    "ollama",
		Options: make(map[string]any),
	}
}

// Merge combines the source ProviderConfig into this ProviderConfig.
// Non-empty name, base_url, and options from source override the current values.
func (c *ProviderConfig) Merge(source *ProviderConfig) {
	if source.Name != "" {
		c.Name = source.Name
	}

	if source.BaseURL != "" {
		c.BaseURL = source.BaseURL
	}

	if source.Options != nil {
		if c.Options == nil {
			c.Options = make(map[string]any)
		}
		maps.Copy(c.Options, source.Options)
	}
}
