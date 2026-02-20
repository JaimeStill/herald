package database_test

import (
	"strings"
	"testing"
	"time"

	"github.com/JaimeStill/herald/pkg/database"
)

func TestFinalizeDefaults(t *testing.T) {
	cfg := database.Config{Name: "testdb", User: "testuser"}
	if err := cfg.Finalize(nil); err != nil {
		t.Fatalf("finalize failed: %v", err)
	}

	tests := []struct {
		name     string
		got      any
		expected any
	}{
		{"host", cfg.Host, "localhost"},
		{"port", cfg.Port, 5432},
		{"ssl_mode", cfg.SSLMode, "disable"},
		{"max_open_conns", cfg.MaxOpenConns, 25},
		{"max_idle_conns", cfg.MaxIdleConns, 5},
		{"conn_max_lifetime", cfg.ConnMaxLifetime, "15m"},
		{"conn_timeout", cfg.ConnTimeout, "5s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("got %v, want %v", tt.got, tt.expected)
			}
		})
	}
}

func TestFinalizeEnvOverrides(t *testing.T) {
	t.Setenv("TEST_DB_HOST", "remotehost")
	t.Setenv("TEST_DB_PORT", "5433")
	t.Setenv("TEST_DB_NAME", "envdb")
	t.Setenv("TEST_DB_USER", "envuser")
	t.Setenv("TEST_DB_PASSWORD", "envpass")
	t.Setenv("TEST_DB_SSL_MODE", "require")
	t.Setenv("TEST_DB_MAX_OPEN", "50")
	t.Setenv("TEST_DB_MAX_IDLE", "10")
	t.Setenv("TEST_DB_LIFETIME", "30m")
	t.Setenv("TEST_DB_TIMEOUT", "10s")

	env := &database.Env{
		Host:            "TEST_DB_HOST",
		Port:            "TEST_DB_PORT",
		Name:            "TEST_DB_NAME",
		User:            "TEST_DB_USER",
		Password:        "TEST_DB_PASSWORD",
		SSLMode:         "TEST_DB_SSL_MODE",
		MaxOpenConns:    "TEST_DB_MAX_OPEN",
		MaxIdleConns:    "TEST_DB_MAX_IDLE",
		ConnMaxLifetime: "TEST_DB_LIFETIME",
		ConnTimeout:     "TEST_DB_TIMEOUT",
	}

	cfg := database.Config{}
	if err := cfg.Finalize(env); err != nil {
		t.Fatalf("finalize failed: %v", err)
	}

	tests := []struct {
		name     string
		got      any
		expected any
	}{
		{"host", cfg.Host, "remotehost"},
		{"port", cfg.Port, 5433},
		{"name", cfg.Name, "envdb"},
		{"user", cfg.User, "envuser"},
		{"password", cfg.Password, "envpass"},
		{"ssl_mode", cfg.SSLMode, "require"},
		{"max_open_conns", cfg.MaxOpenConns, 50},
		{"max_idle_conns", cfg.MaxIdleConns, 10},
		{"conn_max_lifetime", cfg.ConnMaxLifetime, "30m"},
		{"conn_timeout", cfg.ConnTimeout, "10s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("got %v, want %v", tt.got, tt.expected)
			}
		})
	}
}

func TestFinalizeValidation(t *testing.T) {
	tests := []struct {
		name    string
		cfg     database.Config
		wantErr string
	}{
		{
			name:    "missing name",
			cfg:     database.Config{User: "testuser"},
			wantErr: "name required",
		},
		{
			name:    "missing user",
			cfg:     database.Config{Name: "testdb"},
			wantErr: "user required",
		},
		{
			name:    "invalid conn_max_lifetime",
			cfg:     database.Config{Name: "testdb", User: "testuser", ConnMaxLifetime: "bad"},
			wantErr: "invalid conn_max_lifetime",
		},
		{
			name:    "invalid conn_timeout",
			cfg:     database.Config{Name: "testdb", User: "testuser", ConnTimeout: "bad"},
			wantErr: "invalid conn_timeout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Finalize(nil)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error %q does not contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestMerge(t *testing.T) {
	base := database.Config{
		Host: "localhost",
		Port: 5432,
		Name: "basedb",
		User: "baseuser",
	}

	overlay := database.Config{
		Host: "remotehost",
		Port: 5433,
		Name: "overlaydb",
	}

	base.Merge(&overlay)

	if base.Host != "remotehost" {
		t.Errorf("host: got %s, want remotehost", base.Host)
	}
	if base.Port != 5433 {
		t.Errorf("port: got %d, want 5433", base.Port)
	}
	if base.Name != "overlaydb" {
		t.Errorf("name: got %s, want overlaydb", base.Name)
	}
	if base.User != "baseuser" {
		t.Errorf("user should remain baseuser, got %s", base.User)
	}
}

func TestMergeZeroValuesPreserved(t *testing.T) {
	base := database.Config{
		Host:         "localhost",
		Port:         5432,
		MaxOpenConns: 25,
	}

	overlay := database.Config{}
	base.Merge(&overlay)

	if base.Host != "localhost" {
		t.Errorf("host should remain localhost, got %s", base.Host)
	}
	if base.Port != 5432 {
		t.Errorf("port should remain 5432, got %d", base.Port)
	}
	if base.MaxOpenConns != 25 {
		t.Errorf("max_open_conns should remain 25, got %d", base.MaxOpenConns)
	}
}

func TestDsn(t *testing.T) {
	cfg := database.Config{
		Host:     "localhost",
		Port:     5432,
		Name:     "testdb",
		User:     "testuser",
		Password: "testpass",
		SSLMode:  "disable",
	}

	dsn := cfg.Dsn()
	expected := "host=localhost port=5432 dbname=testdb user=testuser password=testpass sslmode=disable"

	if dsn != expected {
		t.Errorf("dsn:\ngot  %s\nwant %s", dsn, expected)
	}
}

func TestDurationParsers(t *testing.T) {
	cfg := database.Config{
		ConnMaxLifetime: "15m",
		ConnTimeout:     "5s",
	}

	if d := cfg.ConnMaxLifetimeDuration(); d != 15*time.Minute {
		t.Errorf("conn_max_lifetime: got %v, want 15m", d)
	}
	if d := cfg.ConnTimeoutDuration(); d != 5*time.Second {
		t.Errorf("conn_timeout: got %v, want 5s", d)
	}
}
