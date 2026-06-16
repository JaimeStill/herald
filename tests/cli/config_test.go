package cli_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/JaimeStill/herald/internal/cli"
	"github.com/JaimeStill/herald/pkg/auth"
)

func profileSettings(home string) string {
	return filepath.Join(home, ".herald", cli.BaseConfigFile)
}

func profileSecrets(home string) string {
	return filepath.Join(home, ".herald", cli.SecretsConfigFile)
}

func TestLoadDefaults(t *testing.T) {
	isolate(t)

	s, err := cli.Load(nil)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	if s.API != "http://localhost:8080" {
		t.Errorf("api: got %q, want http://localhost:8080", s.API)
	}
	if s.Output != cli.OutputJSON {
		t.Errorf("output: got %q, want json", s.Output)
	}
	if s.Timeout != "10m" {
		t.Errorf("timeout: got %q, want 10m", s.Timeout)
	}
	if s.Auth.Mode != auth.ModeNone {
		t.Errorf("auth mode: got %q, want none", s.Auth.Mode)
	}
}

func TestLoadProfile(t *testing.T) {
	home, _ := isolate(t)
	writeFile(t, profileSettings(home), `{"api":"https://profile.example","output":"jsonl"}`)

	s, err := cli.Load(nil)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	if s.API != "https://profile.example" {
		t.Errorf("api: got %q, want https://profile.example", s.API)
	}
	if s.Output != cli.OutputJSONL {
		t.Errorf("output: got %q, want jsonl", s.Output)
	}
}

func TestLoadLocalOverridesProfile(t *testing.T) {
	home, work := isolate(t)
	writeFile(t, profileSettings(home), `{"api":"https://profile.example","timeout":"5m"}`)
	writeFile(t, filepath.Join(work, cli.BaseConfigFile), `{"api":"https://local.example"}`)

	s, err := cli.Load(nil)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	if s.API != "https://local.example" {
		t.Errorf("api: got %q, want https://local.example (local wins)", s.API)
	}
	if s.Timeout != "5m" {
		t.Errorf("timeout: got %q, want 5m (retained from profile)", s.Timeout)
	}
}

func TestLoadSecretsLayering(t *testing.T) {
	home, work := isolate(t)
	writeFile(t, profileSecrets(home), `{"auth":{"client_secret":"profile-secret"}}`)
	writeFile(t, filepath.Join(work, cli.BaseConfigFile), `{"auth":{"auth_mode":"azure","client_id":"CID"}}`)

	s, err := cli.Load(nil)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	if s.Auth.Mode != auth.ModeAzure {
		t.Errorf("auth mode: got %q, want azure", s.Auth.Mode)
	}
	if s.Auth.ClientID != "CID" {
		t.Errorf("client id: got %q, want CID", s.Auth.ClientID)
	}
	if s.Auth.ClientSecret != "profile-secret" {
		t.Errorf("client secret: got %q, want profile-secret", s.Auth.ClientSecret)
	}
}

func TestLoadOverlayWorkingDirOnly(t *testing.T) {
	_, work := isolate(t)
	t.Setenv("HERALD_CLI_ENV", "auth")
	writeFile(t, filepath.Join(work, cli.BaseConfigFile), `{"api":"https://base.example"}`)
	writeFile(t, filepath.Join(work, "settings.auth.json"), `{"auth":{"auth_mode":"azure","client_id":"CID"}}`)

	s, err := cli.Load(nil)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	if s.Auth.Mode != auth.ModeAzure {
		t.Errorf("auth mode: got %q, want azure (from overlay)", s.Auth.Mode)
	}
	if s.API != "https://base.example" {
		t.Errorf("api: got %q, want https://base.example (from base)", s.API)
	}
}

func TestLoadEnvOverride(t *testing.T) {
	isolate(t)
	t.Setenv("HERALD_CLI_API", "https://env.example")

	s, err := cli.Load(nil)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	if s.API != "https://env.example" {
		t.Errorf("api: got %q, want https://env.example", s.API)
	}
}

func TestLoadFlagOverridesEnv(t *testing.T) {
	isolate(t)
	t.Setenv("HERALD_CLI_API", "https://env.example")

	s, err := cli.Load(&cli.Settings{API: "https://flag.example"})
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	if s.API != "https://flag.example" {
		t.Errorf("api: got %q, want https://flag.example (flag wins)", s.API)
	}
}

func TestScopeDerivation(t *testing.T) {
	t.Run("derived from client id", func(t *testing.T) {
		home, _ := isolate(t)
		writeFile(t, profileSettings(home), `{"auth":{"auth_mode":"azure","client_id":"CID"}}`)

		s, err := cli.Load(nil)
		if err != nil {
			t.Fatalf("load: %v", err)
		}
		if s.Scope != "api://CID/.default" {
			t.Errorf("scope: got %q, want api://CID/.default", s.Scope)
		}
	})

	t.Run("explicit scope preserved", func(t *testing.T) {
		home, _ := isolate(t)
		writeFile(t, profileSettings(home), `{"scope":"api://CID/access_as_user","auth":{"auth_mode":"azure","client_id":"CID"}}`)

		s, err := cli.Load(nil)
		if err != nil {
			t.Fatalf("load: %v", err)
		}
		if s.Scope != "api://CID/access_as_user" {
			t.Errorf("scope: got %q, want api://CID/access_as_user", s.Scope)
		}
	})
}

func TestLoadValidationErrors(t *testing.T) {
	tests := []struct {
		name    string
		config  string
		wantErr string
	}{
		{"invalid output", `{"output":"yaml"}`, "invalid output"},
		{"invalid timeout", `{"timeout":"nope"}`, "invalid timeout"},
		{"azure without scope", `{"auth":{"auth_mode":"azure"}}`, "scope must be set"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, work := isolate(t)
			writeFile(t, filepath.Join(work, cli.BaseConfigFile), tt.config)

			_, err := cli.Load(nil)
			if err == nil {
				t.Fatalf("expected error containing %q", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error %q does not contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestOutputFormatSet(t *testing.T) {
	tests := []struct {
		in      string
		wantErr bool
	}{
		{"json", false},
		{"jsonl", false},
		{"yaml", true},
		{"", true},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			var o cli.OutputFormat
			err := o.Set(tt.in)
			if tt.wantErr {
				if err == nil {
					t.Errorf("Set(%q): expected error", tt.in)
				}
				return
			}
			if err != nil {
				t.Errorf("Set(%q): unexpected error %v", tt.in, err)
			}
			if string(o) != tt.in {
				t.Errorf("Set(%q): got %q", tt.in, o)
			}
		})
	}
}

func TestProfileDir(t *testing.T) {
	home, _ := isolate(t)

	dir, ok := cli.ProfileDir()
	if !ok {
		t.Fatal("ProfileDir: ok = false, want true")
	}
	if want := filepath.Join(home, ".herald"); dir != want {
		t.Errorf("ProfileDir: got %q, want %q", dir, want)
	}
}
