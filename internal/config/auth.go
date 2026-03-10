package config

import (
	"fmt"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
)

// AgentScope is the OAuth scope for acquiring Azure AI Foundry bearer tokens.
const AgentScope = "https://cognitiveservices.azure.com/.default"

// AuthMode identifies the authentication provider for Azure service connections.
type AuthMode string

const (
	// AuthModeNone disables credential creation. Infrastructure.Credential is nil.
	AuthModeNone AuthMode = "none"
	// AuthModeAzure enables Azure identity credentials via the azidentity SDK.
	AuthModeAzure AuthMode = "azure"
)
const (
	EnvAuthMode            = "HERALD_AUTH_MODE"
	EnvAuthManagedIdentity = "HERALD_AUTH_MANAGED_IDENTITY"
	EnvAuthTenantID        = "HERALD_AUTH_TENANT_ID"
	EnvAuthClientID        = "HERALD_AUTH_CLIENT_ID"
	EnvAuthClientSecret    = "HERALD_AUTH_CLIENT_SECRET"
)

// AuthConfig holds Azure identity credential parameters.
// When Mode is "none" (default), no credential is created and all existing
// behavior is preserved. When Mode is "azure", a TokenCredential is created
// using either explicit service principal fields or the DefaultAzureCredential chain.
type AuthConfig struct {
	Mode            AuthMode `json:"auth_mode"`
	ManagedIdentity bool     `json:"managed_identity"`
	TenantID        string   `json:"tenant_id"`
	ClientID        string   `json:"client_id"`
	ClientSecret    string   `json:"client_secret"`
}

// Finalize applies defaults, environment variable overrides, and validation.
func (c *AuthConfig) Finalize() error {
	c.loadDefaults()
	c.loadEnv()
	return c.validate()
}

// Merge overwrites non-zero fields from overlay.
func (c *AuthConfig) Merge(overlay *AuthConfig) {
	if overlay.Mode != "" {
		c.Mode = overlay.Mode
	}
	if overlay.ManagedIdentity {
		c.ManagedIdentity = true
	}
	if overlay.TenantID != "" {
		c.TenantID = overlay.TenantID
	}
	if overlay.ClientID != "" {
		c.ClientID = overlay.ClientID
	}
	if overlay.ClientSecret != "" {
		c.ClientSecret = overlay.ClientSecret
	}
}

// TokenCredential returns a credential based on the configured auth mode.
// Returns nil for "none" mode. For "azure" mode, returns a ClientSecretCredential
// when TenantID, ClientID, and ClientSecret are all set, otherwise falls back to
// DefaultAzureCredential which walks the full Azure credential chain.
func (c *AuthConfig) TokenCredential() (azcore.TokenCredential, error) {
	switch c.Mode {
	case AuthModeNone:
		return nil, nil
	case AuthModeAzure:
		return c.azureCredential()
	default:
		return nil, fmt.Errorf("unsupported auth mode: %s", c.Mode)
	}
}

func (c *AuthConfig) azureCredential() (azcore.TokenCredential, error) {
	if c.TenantID != "" && c.ClientID != "" && c.ClientSecret != "" {
		cred, err := azidentity.NewClientSecretCredential(
			c.TenantID, c.ClientID, c.ClientSecret, nil,
		)
		if err != nil {
			return nil, fmt.Errorf("create client secret credential: %w", err)
		}
		return cred, nil
	}

	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, fmt.Errorf("create default azure credential: %w", err)
	}
	return cred, nil
}

func (c *AuthConfig) loadDefaults() {
	if c.Mode == "" {
		c.Mode = AuthModeNone
	}
}

func (c *AuthConfig) loadEnv() {
	if v := os.Getenv(EnvAuthMode); v != "" {
		c.Mode = AuthMode(v)
	}
	if v := os.Getenv(EnvAuthManagedIdentity); v == "true" || v == "1" {
		c.ManagedIdentity = true
	}
	if v := os.Getenv(EnvAuthTenantID); v != "" {
		c.TenantID = v
	}
	if v := os.Getenv(EnvAuthClientID); v != "" {
		c.ClientID = v
	}
	if v := os.Getenv(EnvAuthClientSecret); v != "" {
		c.ClientSecret = v
	}
}

func (c *AuthConfig) validate() error {
	switch c.Mode {
	case AuthModeNone, AuthModeAzure:
		return nil
	default:
		return fmt.Errorf(
			"invalid auth_mode %q: must be %q or %q",
			c.Mode, AuthModeNone, AuthModeAzure,
		)
	}
}
