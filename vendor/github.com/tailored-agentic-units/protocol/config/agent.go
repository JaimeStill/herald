package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// AgentConfig defines the complete configuration for an agent.
// It includes the agent name, optional system prompt, wire format selection,
// optional client settings, provider configuration, and model configuration.
type AgentConfig struct {
	Name         string          `json:"name"`
	SystemPrompt string          `json:"system_prompt,omitempty"`
	Format       string          `json:"format,omitempty"`
	Client       *ClientConfig   `json:"client,omitempty"`
	Provider     *ProviderConfig `json:"provider"`
	Model        *ModelConfig    `json:"model"`
}

// DefaultAgentConfig creates an AgentConfig with default values.
func DefaultAgentConfig() AgentConfig {
	return AgentConfig{
		Name:         "default-agent",
		SystemPrompt: "",
		Format:       "openai",
		Client:       DefaultClientConfig(),
		Provider:     DefaultProviderConfig(),
		Model:        DefaultModelConfig(),
	}
}

// Merge combines the source AgentConfig into this AgentConfig.
// Non-empty values from source override the current values.
func (c *AgentConfig) Merge(source *AgentConfig) {
	if source.Name != "" {
		c.Name = source.Name
	}

	if source.SystemPrompt != "" {
		c.SystemPrompt = source.SystemPrompt
	}

	if source.Format != "" {
		c.Format = source.Format
	}

	if source.Client != nil {
		if c.Client == nil {
			c.Client = source.Client
		} else {
			c.Client.Merge(source.Client)
		}
	}

	if source.Provider != nil {
		if c.Provider == nil {
			c.Provider = source.Provider
		} else {
			c.Provider.Merge(source.Provider)
		}
	}

	if source.Model != nil {
		if c.Model == nil {
			c.Model = source.Model
		} else {
			c.Model.Merge(source.Model)
		}
	}
}

// LoadAgentConfig loads an AgentConfig from a JSON file and merges it with defaults.
// Returns an error if the file cannot be read or the JSON is invalid.
func LoadAgentConfig(filename string) (*AgentConfig, error) {
	config := DefaultAgentConfig()

	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var loaded AgentConfig
	if err := json.Unmarshal(data, &loaded); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	config.Merge(&loaded)

	return &config, nil
}
