package config

import (
	"fmt"
	"os"

	gaconfig "github.com/JaimeStill/go-agents/pkg/config"
)

const (
	EnvAgentProviderName = "HERALD_AGENT_PROVIDER_NAME"
	EnvAgentBaseURL      = "HERALD_AGENT_BASE_URL"
	EnvAgentToken        = "HERALD_AGENT_TOKEN"
	EnvAgentDeployment   = "HERALD_AGENT_DEPLOYMENT"
	EnvAgentAPIVersion   = "HERALD_AGENT_API_VERSION"
	EnvAgentAuthType     = "HERALD_AGENT_AUTH_TYPE"
	EnvAgentModelName    = "HERALD_AGENT_MODEL_NAME"
)

// FinalizeAgent applies Herald's three-phase finalize pattern to a go-agents AgentConfig:
// defaults from go-agents DefaultAgentConfig, environment variable overrides, and validation.
func FinalizeAgent(c *gaconfig.AgentConfig) error {
	loadAgentDefaults(c)
	loadAgentEnv(c)
	return validateAgent(c)
}

func loadAgentDefaults(c *gaconfig.AgentConfig) {
	defaults := gaconfig.DefaultAgentConfig()
	defaults.Merge(c)
	*c = defaults
}

func loadAgentEnv(c *gaconfig.AgentConfig) {
	if c.Provider == nil {
		c.Provider = &gaconfig.ProviderConfig{}
	}
	if c.Provider.Options == nil {
		c.Provider.Options = make(map[string]any)
	}
	if c.Model == nil {
		c.Model = &gaconfig.ModelConfig{}
	}
	if v := os.Getenv(EnvAgentProviderName); v != "" {
		c.Provider.Name = v
	}
	if v := os.Getenv(EnvAgentBaseURL); v != "" {
		c.Provider.BaseURL = v
	}
	if v := os.Getenv(EnvAgentModelName); v != "" {
		c.Model.Name = v
	}

	setOption := func(envVar, key string) {
		if v := os.Getenv(envVar); v != "" {
			c.Provider.Options[key] = v
		}
	}

	setOption(EnvAgentToken, "token")
	setOption(EnvAgentDeployment, "deployment")
	setOption(EnvAgentAPIVersion, "api_version")
	setOption(EnvAgentAuthType, "auth_type")
}

func validateAgent(c *gaconfig.AgentConfig) error {
	if c.Name == "" {
		return fmt.Errorf("name required")
	}
	if c.Provider == nil {
		return fmt.Errorf("provider required")
	}
	if c.Provider.Name == "" {
		return fmt.Errorf("provider name required")
	}
	if c.Model == nil {
		return fmt.Errorf("model required")
	}
	return nil
}
