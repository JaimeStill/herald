// Package cli implements the herald command-line client: a stateless,
// Azure-CLI-style tool for bulk document upload, classification, and result
// retrieval against the Herald API. Commands take inputs from flags or stdin and
// emit JSON to stdout; batch commands own their concurrency, client-side rate
// limiting, and retry/backoff so callers cannot overload the API with a naive
// shell loop.
package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
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

// OutputFormat is the serialization format for command results written to
// stdout.
type OutputFormat string

const (
	// OutputJSON emits results as a single indented JSON value (an array for
	// batch commands).
	OutputJSON OutputFormat = "json"
	// OutputJSONL emits one compact JSON value per line, suited to streaming
	// large batches into line-oriented consumers.
	OutputJSONL OutputFormat = "jsonl"
)

// Built-in defaults applied by loadDefaults before env and flag overrides.
const (
	defaultAPI        = "http://localhost:8080"
	defaultMaxRetries = 5
	defaultTimeout    = "10m"
	defaultOutput     = OutputJSON
)

// Env maps Settings fields to their environment variable names. All CLI
// variables use the HERALD_CLI_ prefix so they never collide with the server's
// HERALD_ variables.
type Env struct {
	API         string
	Scope       string
	Concurrency string
	Rate        string
	Burst       string
	MaxRetries  string
	Timeout     string
	Output      string
}

// settingsEnv is the canonical field-to-variable mapping for top-level settings.
var settingsEnv = &Env{
	API:         "HERALD_CLI_API",
	Scope:       "HERALD_CLI_SCOPE",
	Concurrency: "HERALD_CLI_CONCURRENCY",
	Rate:        "HERALD_CLI_RATE",
	Burst:       "HERALD_CLI_BURST",
	MaxRetries:  "HERALD_CLI_MAX_RETRIES",
	Timeout:     "HERALD_CLI_TIMEOUT",
	Output:      "HERALD_CLI_OUTPUT",
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

// Settings is the resolved configuration for a herald CLI invocation. The
// throttle fields (Concurrency, Rate, Burst) left at zero defer to each command's
// own conservative default, applied where the command builds its batch. Timeout
// is a duration string (e.g. "10m") to match the server's config conventions.
type Settings struct {
	API         string       `json:"api"`
	Scope       string       `json:"scope"`
	Concurrency int          `json:"concurrency"`
	Rate        float64      `json:"rate"`
	Burst       int          `json:"burst"`
	MaxRetries  int          `json:"max_retries"`
	Timeout     string       `json:"timeout"`
	Output      OutputFormat `json:"output"`
	Auth        auth.Config  `json:"auth"`
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
	if overlay.Concurrency != 0 {
		s.Concurrency = overlay.Concurrency
	}
	if overlay.Rate != 0 {
		s.Rate = overlay.Rate
	}
	if overlay.Burst != 0 {
		s.Burst = overlay.Burst
	}
	if overlay.MaxRetries != 0 {
		s.MaxRetries = overlay.MaxRetries
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
	if s.MaxRetries == 0 {
		s.MaxRetries = defaultMaxRetries
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
	if v := os.Getenv(env.Concurrency); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			s.Concurrency = n
		}
	}
	if v := os.Getenv(env.Rate); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			s.Rate = f
		}
	}
	if v := os.Getenv(env.Burst); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			s.Burst = n
		}
	}
	if v := os.Getenv(env.MaxRetries); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			s.MaxRetries = n
		}
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
	if s.MaxRetries < 0 {
		return fmt.Errorf("max_retries must be >= 0")
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
