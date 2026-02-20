package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/JaimeStill/herald/internal/config"
)

const baseConfig = `
shutdown_timeout = "30s"
version = "0.1.0"

[server]
host = "0.0.0.0"
port = 8080
read_timeout = "1m"
write_timeout = "15m"
shutdown_timeout = "30s"

[database]
host = "localhost"
port = 5432
name = "herald"
user = "herald"
password = "herald"
ssl_mode = "disable"
max_open_conns = 25
max_idle_conns = 5
conn_max_lifetime = "15m"
conn_timeout = "5s"

[storage]
container_name = "documents"
connection_string = "DefaultEndpointsProtocol=http;AccountName=heraldstore;AccountKey=key;BlobEndpoint=http://127.0.0.1:10000/heraldstore;"

[api]
base_path = "/api"

[api.cors]
enabled = false

[api.openapi]
title = "Herald API"
description = "Test description"

[api.pagination]
default_page_size = 25
max_page_size = 50
`

const overlayConfig = `
[server]
port = 9090

[database]
host = "prodhost"
`

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
	writeConfig(t, dir, "config.toml", baseConfig)
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
	writeConfig(t, dir, "config.toml", baseConfig)
	writeConfig(t, dir, "config.staging.toml", overlayConfig)
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
	writeConfig(t, dir, "config.toml", baseConfig)
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
		t.Fatalf("load without config.toml failed: %v", err)
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
	writeConfig(t, dir, "config.toml", "invalid = [toml")
	chdir(t, dir)

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error for invalid TOML")
	}
}

func TestEnvDefault(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, "config.toml", baseConfig)
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
	writeConfig(t, dir, "config.toml", baseConfig)
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
	writeConfig(t, dir, "config.toml", baseConfig)
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
	writeConfig(t, dir, "config.toml", baseConfig)
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
	// Config with no pagination section — defaults should apply
	writeConfig(t, dir, "config.toml", `
shutdown_timeout = "30s"
[server]
port = 8080
[database]
name = "herald"
user = "herald"
[storage]
connection_string = "conn"
[api]
base_path = "/api"
`)
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
	writeConfig(t, dir, "config.toml", baseConfig)
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

func TestServerValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  string
		wantErr string
	}{
		{
			name: "invalid port",
			config: `
shutdown_timeout = "30s"
[server]
port = 99999
[database]
name = "herald"
user = "herald"
[storage]
connection_string = "conn"
`,
			wantErr: "invalid port",
		},
		{
			name: "invalid read_timeout",
			config: `
shutdown_timeout = "30s"
[server]
read_timeout = "bad"
[database]
name = "herald"
user = "herald"
[storage]
connection_string = "conn"
`,
			wantErr: "invalid read_timeout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeConfig(t, dir, "config.toml", tt.config)
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
