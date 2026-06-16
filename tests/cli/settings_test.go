package cli_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/JaimeStill/herald/internal/cli"
)

// readJSON reads path and unmarshals it into a generic object.
func readJSON(t *testing.T, path string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	return doc
}

func authSection(t *testing.T, doc map[string]any) map[string]any {
	t.Helper()
	auth, ok := doc["auth"].(map[string]any)
	if !ok {
		t.Fatalf("missing auth section in %v", doc)
	}
	return auth
}

func TestSettingsSecretArg(t *testing.T) {
	home, _ := isolate(t)

	if err := cli.Run([]string{"settings", "secret", "my-secret"}); err != nil {
		t.Fatalf("run: %v", err)
	}

	path := filepath.Join(home, ".herald", cli.SecretsConfigFile)
	if got := authSection(t, readJSON(t, path))["client_secret"]; got != "my-secret" {
		t.Errorf("client_secret: got %v, want my-secret", got)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("perms: got %o, want 600", perm)
	}
}

func TestSettingsSecretStdin(t *testing.T) {
	home, _ := isolate(t)

	withStdin(t, "piped-secret\n", func() {
		if err := cli.Run([]string{"settings", "secret"}); err != nil {
			t.Fatalf("run: %v", err)
		}
	})

	path := filepath.Join(home, ".herald", cli.SecretsConfigFile)
	if got := authSection(t, readJSON(t, path))["client_secret"]; got != "piped-secret" {
		t.Errorf("client_secret: got %v, want piped-secret (trimmed)", got)
	}
}

func TestSettingsSecretPreservesKeys(t *testing.T) {
	home, _ := isolate(t)
	path := filepath.Join(home, ".herald", cli.SecretsConfigFile)
	writeFile(t, path, `{"api":"https://x.example","auth":{"tenant_id":"T"}}`)

	if err := cli.Run([]string{"settings", "secret", "added"}); err != nil {
		t.Fatalf("run: %v", err)
	}

	doc := readJSON(t, path)
	if got := doc["api"]; got != "https://x.example" {
		t.Errorf("api: got %v, want https://x.example (preserved)", got)
	}
	auth := authSection(t, doc)
	if got := auth["tenant_id"]; got != "T" {
		t.Errorf("tenant_id: got %v, want T (preserved)", got)
	}
	if got := auth["client_secret"]; got != "added" {
		t.Errorf("client_secret: got %v, want added", got)
	}
}

// resolvedSecret runs `settings show` (with extra args) and returns the resolved
// auth.client_secret from the emitted JSON.
func resolvedSecret(t *testing.T, args ...string) string {
	t.Helper()
	out := captureStdout(t, func() {
		if err := cli.Run(append([]string{"settings", "show"}, args...)); err != nil {
			t.Fatalf("run: %v", err)
		}
	})
	var got struct {
		Auth struct {
			ClientSecret string `json:"client_secret"`
		} `json:"auth"`
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("parse output %q: %v", out, err)
	}
	return got.Auth.ClientSecret
}

func TestSettingsShowRedacted(t *testing.T) {
	home, _ := isolate(t)
	writeFile(t, profileSettings(home), `{"auth":{"auth_mode":"azure","client_id":"CID"}}`)
	writeFile(t, profileSecrets(home), `{"auth":{"client_secret":"top-secret"}}`)

	if got := resolvedSecret(t); got != "<redacted>" {
		t.Errorf("client_secret: got %q, want <redacted>", got)
	}
}

func TestSettingsShowRevealed(t *testing.T) {
	home, _ := isolate(t)
	writeFile(t, profileSettings(home), `{"auth":{"auth_mode":"azure","client_id":"CID"}}`)
	writeFile(t, profileSecrets(home), `{"auth":{"client_secret":"top-secret"}}`)

	if got := resolvedSecret(t, "--show-secrets"); got != "top-secret" {
		t.Errorf("client_secret: got %q, want top-secret", got)
	}
}

func TestSettingsShowNoSecret(t *testing.T) {
	isolate(t)

	if got := resolvedSecret(t); got != "" {
		t.Errorf("client_secret: got %q, want empty (unset is not redacted)", got)
	}
}
