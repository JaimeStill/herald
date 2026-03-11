package config_test

import (
	"strings"
	"testing"

	"github.com/JaimeStill/herald/internal/config"
	"github.com/JaimeStill/herald/pkg/auth"
)

func TestAuthConfigDefaults(t *testing.T) {
	cfg := &auth.Config{}
	if err := cfg.Finalize(nil); err != nil {
		t.Fatalf("finalize failed: %v", err)
	}

	if cfg.Mode != auth.ModeNone {
		t.Errorf("mode: got %q, want %q", cfg.Mode, auth.ModeNone)
	}
	if cfg.ManagedIdentity {
		t.Error("managed_identity should default to false")
	}
}

func TestAuthConfigNoneCredential(t *testing.T) {
	cfg := &auth.Config{}
	if err := cfg.Finalize(nil); err != nil {
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
		mode    auth.Mode
		wantErr string
	}{
		{"none is valid", auth.ModeNone, ""},
		{"azure is valid", auth.ModeAzure, ""},
		{"invalid mode", "bad", "invalid auth_mode"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &auth.Config{Mode: tt.mode}
			err := cfg.Finalize(nil)

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
	t.Setenv("HERALD_AUTH_MANAGED_IDENTITY", "true")
	t.Setenv("HERALD_AUTH_TENANT_ID", "tenant-123")
	t.Setenv("HERALD_AUTH_CLIENT_ID", "client-456")
	t.Setenv("HERALD_AUTH_CLIENT_SECRET", "secret-789")

	env := &auth.Env{
		Mode:            "HERALD_AUTH_MODE",
		ManagedIdentity: "HERALD_AUTH_MANAGED_IDENTITY",
		TenantID:        "HERALD_AUTH_TENANT_ID",
		ClientID:        "HERALD_AUTH_CLIENT_ID",
		ClientSecret:    "HERALD_AUTH_CLIENT_SECRET",
	}

	cfg := &auth.Config{}
	if err := cfg.Finalize(env); err != nil {
		t.Fatalf("finalize failed: %v", err)
	}

	if cfg.Mode != auth.ModeAzure {
		t.Errorf("mode: got %q, want %q", cfg.Mode, auth.ModeAzure)
	}
	if !cfg.ManagedIdentity {
		t.Error("managed_identity: got false, want true")
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

func TestAuthConfigManagedIdentityEnvValues(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{"true string", "true", true},
		{"1 string", "1", true},
		{"false string", "false", false},
		{"0 string", "0", false},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != "" {
				t.Setenv("HERALD_AUTH_MANAGED_IDENTITY", tt.value)
			}

			env := &auth.Env{
				ManagedIdentity: "HERALD_AUTH_MANAGED_IDENTITY",
			}

			cfg := &auth.Config{}
			if err := cfg.Finalize(env); err != nil {
				t.Fatalf("finalize failed: %v", err)
			}

			if cfg.ManagedIdentity != tt.want {
				t.Errorf("managed_identity: got %v, want %v", cfg.ManagedIdentity, tt.want)
			}
		})
	}
}

func TestAuthConfigMerge(t *testing.T) {
	base := &auth.Config{
		Mode:     auth.ModeNone,
		TenantID: "base-tenant",
	}

	overlay := &auth.Config{
		Mode:     auth.ModeAzure,
		ClientID: "overlay-client",
	}

	base.Merge(overlay)

	if base.Mode != auth.ModeAzure {
		t.Errorf("mode: got %q, want %q", base.Mode, auth.ModeAzure)
	}
	if base.TenantID != "base-tenant" {
		t.Errorf("tenant_id: got %q, want %q (preserved from base)", base.TenantID, "base-tenant")
	}
	if base.ClientID != "overlay-client" {
		t.Errorf("client_id: got %q, want %q (from overlay)", base.ClientID, "overlay-client")
	}
}

func TestAuthConfigMergeManagedIdentity(t *testing.T) {
	base := &auth.Config{Mode: auth.ModeAzure}
	overlay := &auth.Config{ManagedIdentity: true}

	base.Merge(overlay)

	if !base.ManagedIdentity {
		t.Error("managed_identity should be true after merge")
	}
}

func TestAuthConfigMergeManagedIdentityFalsePreserves(t *testing.T) {
	base := &auth.Config{
		Mode:            auth.ModeAzure,
		ManagedIdentity: true,
	}
	overlay := &auth.Config{}

	base.Merge(overlay)

	if !base.ManagedIdentity {
		t.Error("managed_identity should be preserved when overlay is false")
	}
}

func TestAuthConfigMergeEmptyPreserves(t *testing.T) {
	base := &auth.Config{
		Mode:            auth.ModeAzure,
		ManagedIdentity: true,
		TenantID:        "tenant",
		ClientID:        "client",
		ClientSecret:    "secret",
	}

	overlay := &auth.Config{}
	base.Merge(overlay)

	if base.Mode != auth.ModeAzure {
		t.Errorf("mode should be preserved: got %q", base.Mode)
	}
	if !base.ManagedIdentity {
		t.Error("managed_identity should be preserved")
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

	if cfg.Auth.Mode != auth.ModeNone {
		t.Errorf("auth mode: got %q, want %q", cfg.Auth.Mode, auth.ModeNone)
	}
}

func TestAuthConfigFromLoadWithOverlay(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, "config.json", baseConfig)
	writeConfig(t, dir, "config.staging.json", `{
		"auth": {
			"auth_mode": "azure",
			"managed_identity": true,
			"tenant_id": "staging-tenant"
		}
	}`)
	chdir(t, dir)

	t.Setenv("HERALD_ENV", "staging")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	if cfg.Auth.Mode != auth.ModeAzure {
		t.Errorf("auth mode: got %q, want %q", cfg.Auth.Mode, auth.ModeAzure)
	}
	if !cfg.Auth.ManagedIdentity {
		t.Error("managed_identity: got false, want true (from overlay)")
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
	cfg := &auth.Config{Mode: "unsupported"}
	_, err := cfg.TokenCredential()
	if err == nil {
		t.Fatal("expected error for unsupported mode")
	}
	if !strings.Contains(err.Error(), "unsupported auth mode") {
		t.Errorf("error %q does not contain %q", err.Error(), "unsupported auth mode")
	}
}
