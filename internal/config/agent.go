package config

import (
	"encoding/json"
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

	// EnvAgentCapabilitiesChat and EnvAgentCapabilitiesVision carry a full
	// per-capability options object as a JSON string (e.g.
	// {"max_completion_tokens":4096,"reasoning_effort":"high"}). The parsed object
	// replaces the corresponding capabilities block, splatting verbatim into the
	// provider request — keeping the option set model-agnostic (no option keys are
	// hardcoded in Herald). These exist because the deployed container is
	// config-file-free and capabilities are not otherwise reachable via env vars.
	EnvAgentCapabilitiesChat   = "HERALD_AGENT_CAPABILITIES_CHAT"
	EnvAgentCapabilitiesVision = "HERALD_AGENT_CAPABILITIES_VISION"
)

// FinalizeAgent applies Herald's three-phase finalize pattern to a tau AgentConfig:
// defaults from tau protocol/config DefaultAgentConfig, environment variable overrides,
// and validation.
func FinalizeAgent(c *tauconfig.AgentConfig) error {
	loadAgentDefaults(c)
	if err := loadAgentEnv(c); err != nil {
		return err
	}
	return validateAgent(c)
}

func loadAgentDefaults(c *tauconfig.AgentConfig) {
	defaults := tauconfig.DefaultAgentConfig()
	defaults.Merge(c)
	*c = defaults
}

func loadAgentEnv(c *tauconfig.AgentConfig) error {
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

	if err := setCapability(c, EnvAgentCapabilitiesChat, "chat"); err != nil {
		return err
	}
	if err := setCapability(c, EnvAgentCapabilitiesVision, "vision"); err != nil {
		return err
	}

	return nil
}

// setCapability replaces a model capability block from a JSON-object env var,
// leaving the configured value untouched when the variable is unset. The parsed
// object is splatted verbatim into the provider request, so Herald never needs to
// know which option keys a given model supports.
func setCapability(c *tauconfig.AgentConfig, envVar, capability string) error {
	v := os.Getenv(envVar)
	if v == "" {
		return nil
	}

	var options map[string]any
	if err := json.Unmarshal([]byte(v), &options); err != nil {
		return fmt.Errorf("%s: %w", envVar, err)
	}

	if c.Model.Capabilities == nil {
		c.Model.Capabilities = make(map[string]map[string]any)
	}
	c.Model.Capabilities[capability] = options

	return nil
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
