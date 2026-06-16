package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// redactedSecret masks a set client secret in settings show output.
const redactedSecret = "<redacted>"

// settingsShow emits the fully resolved settings from all combined sources
// (profile, working directory, environment, and flags), reflecting the directory
// the command runs from. The client secret is redacted unless --show-secrets is
// passed; the field stays visible so it is clear a secret is configured.
func settingsShow(_ context.Context, args []string) error {
	fs := flag.NewFlagSet("settings show", flag.ContinueOnError)
	flags := bindFlags(fs)
	showSecrets := fs.Bool("show-secrets", false, "reveal secret values instead of redacting them")
	if err := fs.Parse(args); err != nil {
		return err
	}

	s, err := Load(flags)
	if err != nil {
		return err
	}

	view := *s
	if !*showSecrets {
		redactSecrets(&view)
	}
	return emit(s.Output, view)
}

// redactSecrets masks every populated secret field in s so resolved settings can
// be displayed without exposing credentials. Empty fields are left untouched so it
// stays clear which secrets are unset. Extend this as new secret fields are added.
func redactSecrets(s *Settings) {
	if s.Auth.ClientSecret != "" {
		s.Auth.ClientSecret = redactedSecret
	}
}

// settingsSecret writes the Entra client secret to the profile secrets file
// (~/.herald/secrets.json), preserving any other keys already present. The secret
// is read from the positional argument, or from stdin when the argument is "-" or
// omitted, so it can be piped (e.g. from `az keyvault secret show ... -o tsv`).
func settingsSecret(_ context.Context, args []string) error {
	fs := flag.NewFlagSet("settings secret", flag.ContinueOnError)
	if err := fs.Parse(args); err != nil {
		return err
	}

	secret, err := readSecretArg(fs.Arg(0))
	if err != nil {
		return err
	}
	if secret == "" {
		return fmt.Errorf("empty secret")
	}

	dir, ok := ProfileDir()
	if !ok {
		return fmt.Errorf("cannot determine home directory for profile")
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create profile dir: %w", err)
	}

	path := filepath.Join(dir, SecretsConfigFile)
	doc, err := readJSONObject(path)
	if err != nil {
		return err
	}

	authSection, _ := doc["auth"].(map[string]any)
	if authSection == nil {
		authSection = map[string]any{}
	}
	authSection["client_secret"] = secret
	doc["auth"] = authSection

	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("encode secrets: %w", err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
		return fmt.Errorf("write secrets: %w", err)
	}

	fmt.Fprintf(os.Stderr, "wrote client secret to %s\n", path)
	return nil
}

// readSecretArg returns the secret from arg, or reads it from stdin when arg is
// empty or "-". Surrounding whitespace (e.g. a trailing newline from a pipe) is
// trimmed.
func readSecretArg(arg string) (string, error) {
	if arg != "" && arg != "-" {
		return strings.TrimSpace(arg), nil
	}
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return "", fmt.Errorf("read secret from stdin: %w", err)
	}
	return strings.TrimSpace(string(data)), nil
}

// readJSONObject reads path as a JSON object, returning an empty map when the
// file is absent or empty.
func readJSONObject(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]any{}, nil
		}
		return nil, fmt.Errorf("read secrets: %w", err)
	}
	doc := map[string]any{}
	if len(strings.TrimSpace(string(data))) > 0 {
		if err := json.Unmarshal(data, &doc); err != nil {
			return nil, fmt.Errorf("parse secrets: %w", err)
		}
	}
	return doc, nil
}
