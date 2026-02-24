package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/JaimeStill/herald/internal/config"
)

const baseConfig = `{
  "shutdown_timeout": "30s",
  "version": "0.1.0",
  "server": {
    "host": "0.0.0.0",
    "port": 8080,
    "read_timeout": "1m",
    "write_timeout": "15m",
    "shutdown_timeout": "30s"
  },
  "database": {
    "host": "localhost",
    "port": 5432,
    "name": "herald",
    "user": "herald",
    "password": "herald",
    "ssl_mode": "disable",
    "max_open_conns": 25,
    "max_idle_conns": 5,
    "conn_max_lifetime": "15m",
    "conn_timeout": "5s"
  },
  "storage": {
    "container_name": "documents",
    "connection_string": "DefaultEndpointsProtocol=http;AccountName=heraldstore;AccountKey=key;BlobEndpoint=http://127.0.0.1:10000/heraldstore;"
  },
  "api": {
    "base_path": "/api",
    "cors": {
      "enabled": false
    },
    "pagination": {
      "default_page_size": 25,
      "max_page_size": 50
    }
  },
  "agent": {
    "name": "test-agent",
    "provider": {
      "name": "ollama",
      "base_url": "http://localhost:11434"
    },
    "model": {
      "name": "llama3.1:8b"
    }
  }
}`

const overlayConfig = `{
  "server": {
    "port": 9090
  },
  "database": {
    "host": "prodhost"
  }
}`

// minimalConfig provides the minimum fields required
// for validation to pass (db name, db user, storage connection string).
// Agent defaults fill in from go-agents DefaultAgentConfig().
const minimalConfig = `{
  "shutdown_timeout": "30s",
  "server": {
    "port": 8080
  },
  "database": {
    "name": "herald",
    "user": "herald"
  },
  "storage": {
    "connection_string": "conn"
  },
  "api": {
    "base_path": "/api"
  }
}`

func writeConfig(t *testing.T, dir, filename, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, filename), []byte(content), 0644); err != nil {
		t.Fatalf("write %s: %v", filename, err)
	}
}

func chdir(t *testing.T, dir string) {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { os.Chdir(orig) })
}

func TestLoad(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, "config.json", baseConfig)
	chdir(t, dir)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	if cfg.Server.Port != 8080 {
		t.Errorf("server port: got %d, want 8080", cfg.Server.Port)
	}
	if cfg.Database.Host != "localhost" {
		t.Errorf("db host: got %s, want localhost", cfg.Database.Host)
	}
	if cfg.Storage.ContainerName != "documents" {
		t.Errorf("storage container: got %s, want documents", cfg.Storage.ContainerName)
	}
	if cfg.API.BasePath != "/api" {
		t.Errorf("api base_path: got %s, want /api", cfg.API.BasePath)
	}
	if cfg.API.Pagination.DefaultPageSize != 25 {
		t.Errorf("pagination default_page_size: got %d, want 25", cfg.API.Pagination.DefaultPageSize)
	}
	if cfg.API.Pagination.MaxPageSize != 50 {
		t.Errorf("pagination max_page_size: got %d, want 50", cfg.API.Pagination.MaxPageSize)
	}
}

func TestLoadWithOverlay(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, "config.json", baseConfig)
	writeConfig(t, dir, "config.staging.json", overlayConfig)
	chdir(t, dir)

	t.Setenv("HERALD_ENV", "staging")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	if cfg.Server.Port != 9090 {
		t.Errorf("server port: got %d, want 9090 (from overlay)", cfg.Server.Port)
	}
	if cfg.Database.Host != "prodhost" {
		t.Errorf("db host: got %s, want prodhost (from overlay)", cfg.Database.Host)
	}
	if cfg.Database.Port != 5432 {
		t.Errorf("db port: got %d, want 5432 (from base)", cfg.Database.Port)
	}
}

func TestLoadEnvVarOverrides(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, "config.json", baseConfig)
	chdir(t, dir)

	t.Setenv("HERALD_VERSION", "2.0.0")
	t.Setenv("HERALD_SERVER_PORT", "3000")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	if cfg.Version != "2.0.0" {
		t.Errorf("version: got %s, want 2.0.0", cfg.Version)
	}
	if cfg.Server.Port != 3000 {
		t.Errorf("server port: got %d, want 3000", cfg.Server.Port)
	}
}

func TestLoadNoConfigFile(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	t.Setenv("HERALD_DB_NAME", "testdb")
	t.Setenv("HERALD_DB_USER", "testuser")
	t.Setenv("HERALD_STORAGE_CONNECTION_STRING", "conn")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load without config.json failed: %v", err)
	}

	if cfg.Server.Port != 8080 {
		t.Errorf("server port default: got %d, want 8080", cfg.Server.Port)
	}
	if cfg.Database.Name != "testdb" {
		t.Errorf("db name from env: got %s, want testdb", cfg.Database.Name)
	}
	if cfg.Storage.ConnectionString != "conn" {
		t.Errorf("storage conn from env: got %s, want conn", cfg.Storage.ConnectionString)
	}
}

func TestLoadInvalidConfig(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, "config.json", `{"invalid": }`)
	chdir(t, dir)

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestEnvDefault(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, "config.json", baseConfig)
	chdir(t, dir)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	if cfg.Env() != "local" {
		t.Errorf("env: got %s, want local", cfg.Env())
	}
}

func TestEnvFromEnvVar(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, "config.json", baseConfig)
	chdir(t, dir)

	t.Setenv("HERALD_ENV", "production")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	if cfg.Env() != "production" {
		t.Errorf("env: got %s, want production", cfg.Env())
	}
}

func TestShutdownTimeoutDuration(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, "config.json", baseConfig)
	chdir(t, dir)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	if d := cfg.ShutdownTimeoutDuration(); d != 30*time.Second {
		t.Errorf("shutdown timeout: got %v, want 30s", d)
	}
}

func TestServerAddr(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, "config.json", baseConfig)
	chdir(t, dir)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	if addr := cfg.Server.Addr(); addr != "0.0.0.0:8080" {
		t.Errorf("addr: got %s, want 0.0.0.0:8080", addr)
	}
}

func TestPaginationDefaults(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, "config.json", minimalConfig)
	chdir(t, dir)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	if cfg.API.Pagination.DefaultPageSize != 20 {
		t.Errorf("pagination default_page_size: got %d, want 20", cfg.API.Pagination.DefaultPageSize)
	}
	if cfg.API.Pagination.MaxPageSize != 100 {
		t.Errorf("pagination max_page_size: got %d, want 100", cfg.API.Pagination.MaxPageSize)
	}
}

func TestPaginationEnvOverrides(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, "config.json", baseConfig)
	chdir(t, dir)

	t.Setenv("HERALD_PAGINATION_DEFAULT_PAGE_SIZE", "10")
	t.Setenv("HERALD_PAGINATION_MAX_PAGE_SIZE", "200")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	if cfg.API.Pagination.DefaultPageSize != 10 {
		t.Errorf("pagination default_page_size: got %d, want 10", cfg.API.Pagination.DefaultPageSize)
	}
	if cfg.API.Pagination.MaxPageSize != 200 {
		t.Errorf("pagination max_page_size: got %d, want 200", cfg.API.Pagination.MaxPageSize)
	}
}

func TestMaxUploadSizeBytes(t *testing.T) {
	tests := []struct {
		name string
		size string
		want int64
	}{
		{"valid 50MB", "50MB", 50 * 1024 * 1024},
		{"valid 10MB", "10MB", 10 * 1024 * 1024},
		{"valid 1GB", "1GB", 1024 * 1024 * 1024},
		{"invalid falls back to 50MB", "bad", 50 * 1024 * 1024},
		{"empty falls back to 50MB", "", 50 * 1024 * 1024},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.APIConfig{MaxUploadSize: tt.size}
			got := cfg.MaxUploadSizeBytes()
			if got != tt.want {
				t.Errorf("MaxUploadSizeBytes() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestMaxUploadSizeDefault(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, "config.json", baseConfig)
	chdir(t, dir)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	want := int64(50 * 1024 * 1024)
	if got := cfg.API.MaxUploadSizeBytes(); got != want {
		t.Errorf("MaxUploadSizeBytes() = %d, want %d", got, want)
	}
}

func TestMaxUploadSizeEnvOverride(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, "config.json", baseConfig)
	chdir(t, dir)

	t.Setenv("HERALD_API_MAX_UPLOAD_SIZE", "100MB")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	want := int64(100 * 1024 * 1024)
	if got := cfg.API.MaxUploadSizeBytes(); got != want {
		t.Errorf("MaxUploadSizeBytes() = %d, want %d", got, want)
	}
}

func TestServerValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  string
		wantErr string
	}{
		{
			name: "invalid port",
			config: `{
				"shutdown_timeout": "30s",
				"server": {"port": 99999},
				"database": {"name": "herald", "user": "herald"},
				"storage": {"connection_string": "conn"}
			}`,
			wantErr: "invalid port",
		},
		{
			name: "invalid read_timeout",
			config: `{
				"shutdown_timeout": "30s",
				"server": {"read_timeout": "bad"},
				"database": {"name": "herald", "user": "herald"},
				"storage": {"connection_string": "conn"}
			}`,
			wantErr: "invalid read_timeout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeConfig(t, dir, "config.json", tt.config)
			chdir(t, dir)

			_, err := config.Load()
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error %q does not contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestAgentConfigFromJSON(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, "config.json", baseConfig)
	chdir(t, dir)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	if cfg.Agent.Name != "test-agent" {
		t.Errorf("agent name: got %s, want test-agent", cfg.Agent.Name)
	}
	if cfg.Agent.Provider == nil {
		t.Fatal("agent provider is nil")
	}
	if cfg.Agent.Provider.Name != "ollama" {
		t.Errorf("provider name: got %s, want ollama", cfg.Agent.Provider.Name)
	}
	if cfg.Agent.Provider.BaseURL != "http://localhost:11434" {
		t.Errorf("provider base_url: got %s, want http://localhost:11434", cfg.Agent.Provider.BaseURL)
	}
	if cfg.Agent.Model == nil {
		t.Fatal("agent model is nil")
	}
	if cfg.Agent.Model.Name != "llama3.1:8b" {
		t.Errorf("model name: got %s, want llama3.1:8b", cfg.Agent.Model.Name)
	}
}

func TestAgentDefaults(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, "config.json", minimalConfig)
	chdir(t, dir)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	if cfg.Agent.Name != "default-agent" {
		t.Errorf("agent name: got %s, want default-agent", cfg.Agent.Name)
	}
	if cfg.Agent.Provider == nil {
		t.Fatal("agent provider is nil")
	}
	if cfg.Agent.Provider.Name != "ollama" {
		t.Errorf("provider name: got %s, want ollama", cfg.Agent.Provider.Name)
	}
	if cfg.Agent.Provider.BaseURL != "http://localhost:11434" {
		t.Errorf("provider base_url: got %s, want http://localhost:11434", cfg.Agent.Provider.BaseURL)
	}
}

func TestAgentEnvOverrides(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, "config.json", baseConfig)
	chdir(t, dir)

	t.Setenv("HERALD_AGENT_PROVIDER_NAME", "azure")
	t.Setenv("HERALD_AGENT_BASE_URL", "https://myendpoint.openai.azure.com")
	t.Setenv("HERALD_AGENT_MODEL_NAME", "gpt-5-mini")
	t.Setenv("HERALD_AGENT_TOKEN", "test-token")
	t.Setenv("HERALD_AGENT_DEPLOYMENT", "gpt-5-mini")
	t.Setenv("HERALD_AGENT_API_VERSION", "2024-12-01-preview")
	t.Setenv("HERALD_AGENT_AUTH_TYPE", "api_key")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	if cfg.Agent.Provider.Name != "azure" {
		t.Errorf("provider name: got %s, want azure", cfg.Agent.Provider.Name)
	}
	if cfg.Agent.Provider.BaseURL != "https://myendpoint.openai.azure.com" {
		t.Errorf("provider base_url: got %s, want https://myendpoint.openai.azure.com", cfg.Agent.Provider.BaseURL)
	}
	if cfg.Agent.Model.Name != "gpt-5-mini" {
		t.Errorf("model name: got %s, want gpt-5-mini", cfg.Agent.Model.Name)
	}

	opts := cfg.Agent.Provider.Options
	if opts["token"] != "test-token" {
		t.Errorf("token: got %v, want test-token", opts["token"])
	}
	if opts["deployment"] != "gpt-5-mini" {
		t.Errorf("deployment: got %v, want gpt-5-mini", opts["deployment"])
	}
	if opts["api_version"] != "2024-12-01-preview" {
		t.Errorf("api_version: got %v, want 2024-12-01-preview", opts["api_version"])
	}
	if opts["auth_type"] != "api_key" {
		t.Errorf("auth_type: got %v, want api_key", opts["auth_type"])
	}
}

func TestAgentOverlay(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, "config.json", baseConfig)
	writeConfig(t, dir, "config.staging.json", `{
		"agent": {
			"name": "staging-agent",
			"provider": {
				"name": "azure",
				"base_url": "https://staging.openai.azure.com"
			},
			"model": {
				"name": "gpt-5-mini"
			}
		}
	}`)
	chdir(t, dir)

	t.Setenv("HERALD_ENV", "staging")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	if cfg.Agent.Name != "staging-agent" {
		t.Errorf("agent name: got %s, want staging-agent", cfg.Agent.Name)
	}
	if cfg.Agent.Provider.Name != "azure" {
		t.Errorf("provider name: got %s, want azure", cfg.Agent.Provider.Name)
	}
	if cfg.Agent.Provider.BaseURL != "https://staging.openai.azure.com" {
		t.Errorf("provider base_url: got %s, want https://staging.openai.azure.com", cfg.Agent.Provider.BaseURL)
	}
	if cfg.Agent.Model.Name != "gpt-5-mini" {
		t.Errorf("model name: got %s, want gpt-5-mini", cfg.Agent.Model.Name)
	}
	// Base config values should be preserved for non-agent fields
	if cfg.Server.Port != 8080 {
		t.Errorf("server port: got %d, want 8080 (from base)", cfg.Server.Port)
	}
}

func TestAgentTokenNotRequired(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, "config.json", baseConfig)
	chdir(t, dir)

	// No HERALD_AGENT_TOKEN set — should succeed for ollama provider
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	if _, ok := cfg.Agent.Provider.Options["token"]; ok {
		t.Error("token should not be set when env var is absent")
	}
}
