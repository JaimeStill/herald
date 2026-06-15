// Package cli implements the herald command-line client: a stateless,
// Azure-CLI-style primitive over the Herald API. Each command maps to a single
// API call, takes its inputs from flags (or stdin), and emits the response as
// JSON to stdout. Orchestration — batching, concurrency, retries — is left to
// the caller's scripts.
package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/JaimeStill/herald/pkg/auth"
)

// Configuration file names and the overlay-selecting environment variable.
// These deliberately use distinct names from the server (settings.json vs the
// server's config.json) so the CLI can run from the repository root without
// reading the server's base configuration.
const (
	BaseConfigFile       = "settings.json"
	OverlayConfigPattern = "settings.%s.json"
	SecretsConfigFile    = "secrets.json"

	EnvCLIEnv = "HERALD_CLI_ENV"
)

// OutputFormat is the serialization format for a command's result on stdout.
type OutputFormat string

const (
	// OutputJSON emits the result as a single indented JSON value.
	OutputJSON OutputFormat = "json"
	// OutputJSONL emits the result as one compact JSON line, suited to piping
	// into line-oriented consumers.
	OutputJSONL OutputFormat = "jsonl"
)

// String implements flag.Value.
func (o *OutputFormat) String() string { return string(*o) }

// Set implements flag.Value, validating the format at parse time.
func (o *OutputFormat) Set(v string) error {
	switch OutputFormat(v) {
	case OutputJSON, OutputJSONL:
		*o = OutputFormat(v)
		return nil
	default:
		return fmt.Errorf("invalid output %q: must be %q or %q", v, OutputJSON, OutputJSONL)
	}
}

// Built-in defaults applied by loadDefaults before env and flag overrides.
const (
	defaultAPI     = "http://localhost:8080"
	defaultTimeout = "10m"
	defaultOutput  = OutputJSON
)

// Env maps Settings fields to their environment variable names. All CLI
// variables use the HERALD_CLI_ prefix so they never collide with the server's
// HERALD_ variables.
type Env struct {
	API     string
	Scope   string
	Timeout string
	Output  string
}

// settingsEnv is the canonical field-to-variable mapping for top-level settings.
var settingsEnv = &Env{
	API:     "HERALD_CLI_API",
	Scope:   "HERALD_CLI_SCOPE",
	Timeout: "HERALD_CLI_TIMEOUT",
	Output:  "HERALD_CLI_OUTPUT",
}

// authEnv maps the embedded auth.Config to HERALD_CLI_AUTH_* variables, mirroring
// the server's HERALD_AUTH_* layout under the CLI prefix.
var authEnv = &auth.Env{
	Mode:            "HERALD_CLI_AUTH_MODE",
	ManagedIdentity: "HERALD_CLI_AUTH_MANAGED_IDENTITY",
	TenantID:        "HERALD_CLI_AUTH_TENANT_ID",
	ClientID:        "HERALD_CLI_AUTH_CLIENT_ID",
	ClientSecret:    "HERALD_CLI_AUTH_CLIENT_SECRET",
	Authority:       "HERALD_CLI_AUTH_AUTHORITY",
	Scope:           "HERALD_CLI_AUTH_SCOPE",
	CacheLocation:   "HERALD_CLI_AUTH_CACHE_LOCATION",
}

// Settings is the resolved configuration for a herald CLI invocation. Timeout is
// a duration string (e.g. "10m") to match the server's config conventions.
type Settings struct {
	API     string       `json:"api"`
	Scope   string       `json:"scope"`
	Timeout string       `json:"timeout"`
	Output  OutputFormat `json:"output"`
	Auth    auth.Config  `json:"auth"`
}

// Load reads the base settings file (if present), applies any environment
// overlay and secrets file, then finalizes all values with flags as the
// highest-precedence overlay. With no files present, defaults, environment
// variables, and flags provide the entire configuration. flags may be nil.
func Load(flags *Settings) (*Settings, error) {
	s := &Settings{}

	if _, err := os.Stat(BaseConfigFile); err == nil {
		loaded, err := loadFile(BaseConfigFile)
		if err != nil {
			return nil, err
		}
		s = loaded
	}

	if path := overlayPath(); path != "" {
		overlay, err := loadFile(path)
		if err != nil {
			return nil, fmt.Errorf("load overlay %s: %w", path, err)
		}
		s.Merge(overlay)
	}

	if _, err := os.Stat(SecretsConfigFile); err == nil {
		secrets, err := loadFile(SecretsConfigFile)
		if err != nil {
			return nil, err
		}
		s.Merge(secrets)
	}

	if err := s.Finalize(settingsEnv, authEnv, flags); err != nil {
		return nil, fmt.Errorf("finalize settings: %w", err)
	}

	return s, nil
}

// bindFlags registers the global flags on fs, bound into a fresh Settings overlay
// suitable for Load (flags are the highest-precedence overlay). Call it before
// adding command-specific flags and parsing fs, then pass the returned overlay to
// Load. Unset flags stay at their zero value and are ignored by Merge.
func bindFlags(fs *flag.FlagSet) *Settings {
	flags := &Settings{}
	fs.StringVar(&flags.API, "api", "", "Herald API base URL")
	fs.StringVar(&flags.Scope, "scope", "", "Entra token scope (default api://{client-id}/.default)")
	fs.StringVar(&flags.Timeout, "timeout", "", "per-request timeout (e.g. 10m)")
	fs.Var(&flags.Output, "output", "output format: json or jsonl")
	return flags
}

// Finalize applies defaults, environment overrides, derived defaults, the flag
// overlay, and validation, in that order. flags is the highest-precedence
// overlay (CLI flags win over env, which wins over files); it may be nil. The
// auth sub-config is finalized with its own env mapping. Because the overlay is
// merged before validation, the result is validated exactly once.
func (s *Settings) Finalize(env *Env, aenv *auth.Env, flags *Settings) error {
	s.loadDefaults()
	if env != nil {
		s.loadEnv(env)
	}
	if err := s.Auth.Finalize(aenv); err != nil {
		return fmt.Errorf("auth: %w", err)
	}
	s.deriveDefaults()
	if flags != nil {
		s.Merge(flags)
	}
	return s.validate()
}

// Merge overwrites non-zero fields from overlay across all settings, including
// the embedded auth config.
func (s *Settings) Merge(overlay *Settings) {
	if overlay.API != "" {
		s.API = overlay.API
	}
	if overlay.Scope != "" {
		s.Scope = overlay.Scope
	}
	if overlay.Timeout != "" {
		s.Timeout = overlay.Timeout
	}
	if overlay.Output != "" {
		s.Output = overlay.Output
	}
	s.Auth.Merge(&overlay.Auth)
}

// TimeoutDuration returns Timeout parsed as a time.Duration.
func (s *Settings) TimeoutDuration() time.Duration {
	d, _ := time.ParseDuration(s.Timeout)
	return d
}

func (s *Settings) loadDefaults() {
	if s.API == "" {
		s.API = defaultAPI
	}
	if s.Timeout == "" {
		s.Timeout = defaultTimeout
	}
	if s.Output == "" {
		s.Output = defaultOutput
	}
}

func (s *Settings) loadEnv(env *Env) {
	if v := os.Getenv(env.API); v != "" {
		s.API = v
	}
	if v := os.Getenv(env.Scope); v != "" {
		s.Scope = v
	}
	if v := os.Getenv(env.Timeout); v != "" {
		s.Timeout = v
	}
	if v := os.Getenv(env.Output); v != "" {
		s.Output = OutputFormat(v)
	}
}

// deriveDefaults fills the API token scope from the auth client ID when it was
// not set explicitly. The exact scope string is operator-overridable to handle
// IL6 quirks (e.g. trailing-slash and .default vs access_as_user differences).
func (s *Settings) deriveDefaults() {
	if s.Scope == "" && s.Auth.ClientID != "" {
		s.Scope = "api://" + s.Auth.ClientID + "/.default"
	}
}

func (s *Settings) validate() error {
	if s.API == "" {
		return fmt.Errorf("api must be set")
	}
	if _, err := time.ParseDuration(s.Timeout); err != nil {
		return fmt.Errorf("invalid timeout %q: %w", s.Timeout, err)
	}
	switch s.Output {
	case OutputJSON, OutputJSONL:
	default:
		return fmt.Errorf("invalid output %q: must be %q or %q", s.Output, OutputJSON, OutputJSONL)
	}
	if s.Auth.Mode == auth.ModeAzure && s.Scope == "" {
		return fmt.Errorf("scope must be set when auth_mode is azure (set HERALD_CLI_SCOPE)")
	}
	return nil
}

func loadFile(path string) (*Settings, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read settings: %w", err)
	}

	var s Settings
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parse settings: %w", err)
	}

	return &s, nil
}

func overlayPath() string {
	if env := os.Getenv(EnvCLIEnv); env != "" {
		path := fmt.Sprintf(OverlayConfigPattern, env)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}
