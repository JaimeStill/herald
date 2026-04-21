package config

import (
	"fmt"
	"os"

	tauconfig "github.com/tailored-agentic-units/protocol/config"
)

const (
	EnvAgentProviderName = "HERALD_AGENT_PROVIDER_NAME"
	EnvAgentBaseURL      = "HERALD_AGENT_BASE_URL"
	EnvAgentToken        = "HERALD_AGENT_TOKEN"
	EnvAgentDeployment   = "HERALD_AGENT_DEPLOYMENT"
	EnvAgentAPIVersion   = "HERALD_AGENT_API_VERSION"
	EnvAgentAuthType     = "HERALD_AGENT_AUTH_TYPE"
	EnvAgentResource     = "HERALD_AGENT_RESOURCE"
	EnvAgentClientID     = "HERALD_AGENT_CLIENT_ID"
	EnvAgentModelName    = "HERALD_AGENT_MODEL_NAME"
)

// FinalizeAgent applies Herald's three-phase finalize pattern to a tau AgentConfig:
// defaults from tau protocol/config DefaultAgentConfig, environment variable overrides,
// and validation.
func FinalizeAgent(c *tauconfig.AgentConfig) error {
	loadAgentDefaults(c)
	loadAgentEnv(c)
	return validateAgent(c)
}

func loadAgentDefaults(c *tauconfig.AgentConfig) {
	defaults := tauconfig.DefaultAgentConfig()
	defaults.Merge(c)
	*c = defaults
}

func loadAgentEnv(c *tauconfig.AgentConfig) {
	if c.Provider == nil {
		c.Provider = &tauconfig.ProviderConfig{}
	}
	if c.Provider.Options == nil {
		c.Provider.Options = make(map[string]any)
	}
	if c.Model == nil {
		c.Model = &tauconfig.ModelConfig{}
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
	setOption(EnvAgentResource, "resource")
	setOption(EnvAgentClientID, "client_id")
}

func validateAgent(c *tauconfig.AgentConfig) error {
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
