package storage_test

import (
	"strings"
	"testing"

	"github.com/JaimeStill/herald/pkg/storage"
)

func TestFinalizeDefaults(t *testing.T) {
	cfg := storage.Config{ConnectionString: "test-connection"}
	if err := cfg.Finalize(nil); err != nil {
		t.Fatalf("finalize failed: %v", err)
	}

	if cfg.ContainerName != "documents" {
		t.Errorf("container_name: got %s, want documents", cfg.ContainerName)
	}
}

func TestFinalizeEnvOverrides(t *testing.T) {
	t.Setenv("TEST_CONTAINER", "uploads")
	t.Setenv("TEST_CONN", "override-connection")

	env := &storage.Env{
		ContainerName:    "TEST_CONTAINER",
		ConnectionString: "TEST_CONN",
	}

	cfg := storage.Config{}
	if err := cfg.Finalize(env); err != nil {
		t.Fatalf("finalize failed: %v", err)
	}

	if cfg.ContainerName != "uploads" {
		t.Errorf("container_name: got %s, want uploads", cfg.ContainerName)
	}
	if cfg.ConnectionString != "override-connection" {
		t.Errorf("connection_string: got %s, want override-connection", cfg.ConnectionString)
	}
}

func TestFinalizeValidation(t *testing.T) {
	tests := []struct {
		name    string
		cfg     storage.Config
		wantErr string
	}{
		{
			name:    "missing connection_string",
			cfg:     storage.Config{ContainerName: "docs"},
			wantErr: "connection_string required",
		},
		{
			name:    "missing container_name after clearing default",
			cfg:     storage.Config{ConnectionString: "conn"},
			wantErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Finalize(nil)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
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
	base := storage.Config{
		ContainerName:    "documents",
		ConnectionString: "base-conn",
	}

	overlay := storage.Config{ConnectionString: "overlay-conn"}
	base.Merge(&overlay)

	if base.ContainerName != "documents" {
		t.Errorf("container_name should remain documents, got %s", base.ContainerName)
	}
	if base.ConnectionString != "overlay-conn" {
		t.Errorf("connection_string: got %s, want overlay-conn", base.ConnectionString)
	}
}
