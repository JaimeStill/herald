// Package auth provides authentication types, configuration, and request
// context helpers for Azure Entra ID integration.
package auth

import (
	"fmt"
	"os"
	"strconv"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
)

// AgentScope is the OAuth scope for acquiring Azure AI Foundry bearer tokens.
const AgentScope = "https://cognitiveservices.azure.com/.default"

// Mode identifies the authentication strategy for Azure service connections.
type Mode string

const (
	// ModeNone disables authentication. No credentials are created.
	ModeNone Mode = "none"
	// ModeAzure enables Azure identity credentials via the azidentity SDK.
	ModeAzure Mode = "azure"

	// DefaultAuthorityBase is the commercial Azure AD authority URL prefix.
	DefaultAuthorityBase = "https://login.microsoftonline.com/"
	// DefaultAuthorityPath is the OIDC v2.0 endpoint suffix.
	DefaultAuthorityPath = "/v2.0"
)

// Config holds Azure authentication parameters for both infrastructure
// credentials (service-to-service via TokenCredential) and API authentication
// (JWT validation via the Auth middleware). Mode controls both: ModeNone
// disables all auth; ModeAzure enables credential creation and JWT validation.
type Config struct {
	Mode            Mode   `json:"auth_mode"`
	ManagedIdentity bool   `json:"managed_identity"`
	TenantID        string `json:"tenant_id"`
	ClientID        string `json:"client_id"`
	ClientSecret    string `json:"client_secret"`
	Authority       string `json:"authority"`
}

// Env maps Config fields to environment variable names for override injection.
type Env struct {
	Mode            string
	ManagedIdentity string
	TenantID        string
	ClientID        string
	ClientSecret    string
	Authority       string
}

// Finalize applies defaults, environment variable overrides, derived defaults,
// and validation. Authority is derived from TenantID after env overrides when
// not explicitly set.
func (c *Config) Finalize(env *Env) error {
	c.loadDefaults()
	if env != nil {
		c.loadEnv(env)
	}
	c.deriveDefaults()
	return c.validate()
}

// Merge overwrites non-zero fields from overlay. Boolean ManagedIdentity
// only applies when true; string fields apply when non-empty.
func (c *Config) Merge(overlay *Config) {
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
	if overlay.Authority != "" {
		c.Authority = overlay.Authority
	}
}

// TokenCredential returns a credential based on the configured auth mode.
// Returns nil for ModeNone. For ModeAzure, returns a ClientSecretCredential
// when TenantID, ClientID, and ClientSecret are all set, otherwise falls back
// to DefaultAzureCredential which walks the full Azure credential chain.
func (c *Config) TokenCredential() (azcore.TokenCredential, error) {
	switch c.Mode {
	case ModeNone:
		return nil, nil
	case ModeAzure:
		return c.azureCredential()
	default:
		return nil, fmt.Errorf("unsupported auth mode: %s", c.Mode)
	}
}

func (c *Config) azureCredential() (azcore.TokenCredential, error) {
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

func (c *Config) loadDefaults() {
	if c.Mode == "" {
		c.Mode = ModeNone
	}
}

func (c *Config) loadEnv(env *Env) {
	if env.Mode != "" {
		if v := os.Getenv(env.Mode); v != "" {
			c.Mode = Mode(v)
		}
	}
	if env.ManagedIdentity != "" {
		if v := os.Getenv(env.ManagedIdentity); v != "" {
			if b, err := strconv.ParseBool(v); err == nil && b {
				c.ManagedIdentity = true
			}
		}
	}
	if env.TenantID != "" {
		if v := os.Getenv(env.TenantID); v != "" {
			c.TenantID = v
		}
	}
	if env.ClientID != "" {
		if v := os.Getenv(env.ClientID); v != "" {
			c.ClientID = v
		}
	}
	if env.ClientSecret != "" {
		if v := os.Getenv(env.ClientSecret); v != "" {
			c.ClientSecret = v
		}
	}
	if env.Authority != "" {
		if v := os.Getenv(env.Authority); v != "" {
			c.Authority = v
		}
	}
}

func (c *Config) deriveDefaults() {
	if c.Authority == "" && c.TenantID != "" {
		c.Authority = DefaultAuthorityBase + c.TenantID + DefaultAuthorityPath
	}
}

func (c *Config) validate() error {
	switch c.Mode {
	case ModeNone, ModeAzure:
		return nil
	default:
		return fmt.Errorf(
			"invalid auth_mode %q: must be %q or %q",
			c.Mode, ModeNone, ModeAzure,
		)
	}
}
