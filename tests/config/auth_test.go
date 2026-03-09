package config_test

import (
	"strings"
	"testing"

	"github.com/JaimeStill/herald/internal/config"
)

func TestAuthConfigDefaults(t *testing.T) {
	cfg := &config.AuthConfig{}
	if err := cfg.Finalize(); err != nil {
		t.Fatalf("finalize failed: %v", err)
	}

	if cfg.Mode != config.AuthModeNone {
		t.Errorf("mode: got %q, want %q", cfg.Mode, config.AuthModeNone)
	}
}

func TestAuthConfigNoneCredential(t *testing.T) {
	cfg := &config.AuthConfig{}
	if err := cfg.Finalize(); err != nil {
		t.Fatalf("finalize failed: %v", err)
	}

	cred, err := cfg.TokenCredential()
	if err != nil {
		t.Fatalf("TokenCredential failed: %v", err)
	}
	if cred != nil {
		t.Error("expected nil credential for none mode")
	}
}

func TestAuthConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		mode    config.AuthMode
		wantErr string
	}{
		{"none is valid", config.AuthModeNone, ""},
		{"azure is valid", config.AuthModeAzure, ""},
		{"invalid mode", "bad", "invalid auth_mode"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.AuthConfig{Mode: tt.mode}
			err := cfg.Finalize()

			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}

			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error %q does not contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestAuthConfigEnvOverrides(t *testing.T) {
	t.Setenv("HERALD_AUTH_MODE", "azure")
	t.Setenv("HERALD_AUTH_TENANT_ID", "tenant-123")
	t.Setenv("HERALD_AUTH_CLIENT_ID", "client-456")
	t.Setenv("HERALD_AUTH_CLIENT_SECRET", "secret-789")

	cfg := &config.AuthConfig{}
	if err := cfg.Finalize(); err != nil {
		t.Fatalf("finalize failed: %v", err)
	}

	if cfg.Mode != config.AuthModeAzure {
		t.Errorf("mode: got %q, want %q", cfg.Mode, config.AuthModeAzure)
	}
	if cfg.TenantID != "tenant-123" {
		t.Errorf("tenant_id: got %q, want %q", cfg.TenantID, "tenant-123")
	}
	if cfg.ClientID != "client-456" {
		t.Errorf("client_id: got %q, want %q", cfg.ClientID, "client-456")
	}
	if cfg.ClientSecret != "secret-789" {
		t.Errorf("client_secret: got %q, want %q", cfg.ClientSecret, "secret-789")
	}
}

func TestAuthConfigMerge(t *testing.T) {
	base := &config.AuthConfig{
		Mode:     config.AuthModeNone,
		TenantID: "base-tenant",
	}

	overlay := &config.AuthConfig{
		Mode:     config.AuthModeAzure,
		ClientID: "overlay-client",
	}

	base.Merge(overlay)

	if base.Mode != config.AuthModeAzure {
		t.Errorf("mode: got %q, want %q", base.Mode, config.AuthModeAzure)
	}
	if base.TenantID != "base-tenant" {
		t.Errorf("tenant_id: got %q, want %q (preserved from base)", base.TenantID, "base-tenant")
	}
	if base.ClientID != "overlay-client" {
		t.Errorf("client_id: got %q, want %q (from overlay)", base.ClientID, "overlay-client")
	}
}

func TestAuthConfigMergeEmptyPreserves(t *testing.T) {
	base := &config.AuthConfig{
		Mode:         config.AuthModeAzure,
		TenantID:     "tenant",
		ClientID:     "client",
		ClientSecret: "secret",
	}

	overlay := &config.AuthConfig{}
	base.Merge(overlay)

	if base.Mode != config.AuthModeAzure {
		t.Errorf("mode should be preserved: got %q", base.Mode)
	}
	if base.TenantID != "tenant" {
		t.Errorf("tenant_id should be preserved: got %q", base.TenantID)
	}
	if base.ClientID != "client" {
		t.Errorf("client_id should be preserved: got %q", base.ClientID)
	}
	if base.ClientSecret != "secret" {
		t.Errorf("client_secret should be preserved: got %q", base.ClientSecret)
	}
}

func TestAuthConfigFromLoad(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, "config.json", baseConfig)
	chdir(t, dir)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	if cfg.Auth.Mode != config.AuthModeNone {
		t.Errorf("auth mode: got %q, want %q", cfg.Auth.Mode, config.AuthModeNone)
	}
}

func TestAuthConfigFromLoadWithOverlay(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, "config.json", baseConfig)
	writeConfig(t, dir, "config.staging.json", `{
		"auth": {
			"auth_mode": "azure",
			"tenant_id": "staging-tenant"
		}
	}`)
	chdir(t, dir)

	t.Setenv("HERALD_ENV", "staging")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	if cfg.Auth.Mode != config.AuthModeAzure {
		t.Errorf("auth mode: got %q, want %q", cfg.Auth.Mode, config.AuthModeAzure)
	}
	if cfg.Auth.TenantID != "staging-tenant" {
		t.Errorf("tenant_id: got %q, want %q", cfg.Auth.TenantID, "staging-tenant")
	}
}

func TestAuthConfigInvalidModeFromLoad(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, "config.json", `{
		"shutdown_timeout": "30s",
		"auth": {"auth_mode": "bad"},
		"server": {"port": 8080},
		"database": {"name": "herald", "user": "herald"},
		"storage": {"connection_string": "conn"}
	}`)
	chdir(t, dir)

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error for invalid auth_mode")
	}
	if !strings.Contains(err.Error(), "invalid auth_mode") {
		t.Errorf("error %q does not contain %q", err.Error(), "invalid auth_mode")
	}
}

func TestTokenCredentialUnsupportedMode(t *testing.T) {
	cfg := &config.AuthConfig{Mode: "unsupported"}
	_, err := cfg.TokenCredential()
	if err == nil {
		t.Fatal("expected error for unsupported mode")
	}
	if !strings.Contains(err.Error(), "unsupported auth mode") {
		t.Errorf("error %q does not contain %q", err.Error(), "unsupported auth mode")
	}
}
