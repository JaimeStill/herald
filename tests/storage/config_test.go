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
	if cfg.MaxListSize != 50 {
		t.Errorf("max_list_size: got %d, want 50", cfg.MaxListSize)
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

func TestFinalizeMaxListSizeCap(t *testing.T) {
	cfg := storage.Config{
		ConnectionString: "test-connection",
		MaxListSize:      10000,
	}
	if err := cfg.Finalize(nil); err != nil {
		t.Fatalf("finalize failed: %v", err)
	}

	if cfg.MaxListSize != storage.MaxListCap {
		t.Errorf("max_list_size: got %d, want %d (capped)", cfg.MaxListSize, storage.MaxListCap)
	}
}

func TestFinalizeMaxListSizeEnvOverride(t *testing.T) {
	t.Setenv("TEST_MAX_LIST", "200")

	env := &storage.Env{
		ConnectionString: "TEST_CONN",
		MaxListSize:      "TEST_MAX_LIST",
	}

	t.Setenv("TEST_CONN", "test-connection")

	cfg := storage.Config{}
	if err := cfg.Finalize(env); err != nil {
		t.Fatalf("finalize failed: %v", err)
	}

	if cfg.MaxListSize != 200 {
		t.Errorf("max_list_size: got %d, want 200", cfg.MaxListSize)
	}
}

func TestFinalizeMaxListSizeEnvCapped(t *testing.T) {
	t.Setenv("TEST_MAX_LIST", "99999")
	t.Setenv("TEST_CONN", "test-connection")

	env := &storage.Env{
		ConnectionString: "TEST_CONN",
		MaxListSize:      "TEST_MAX_LIST",
	}

	cfg := storage.Config{}
	if err := cfg.Finalize(env); err != nil {
		t.Fatalf("finalize failed: %v", err)
	}

	if cfg.MaxListSize != storage.MaxListCap {
		t.Errorf("max_list_size: got %d, want %d (capped)", cfg.MaxListSize, storage.MaxListCap)
	}
}

func TestMerge(t *testing.T) {
	base := storage.Config{
		ContainerName:    "documents",
		ConnectionString: "base-conn",
		MaxListSize:      50,
	}

	overlay := storage.Config{
		ConnectionString: "overlay-conn",
		MaxListSize:      100,
	}
	base.Merge(&overlay)

	if base.ContainerName != "documents" {
		t.Errorf("container_name should remain documents, got %s", base.ContainerName)
	}
	if base.ConnectionString != "overlay-conn" {
		t.Errorf("connection_string: got %s, want overlay-conn", base.ConnectionString)
	}
	if base.MaxListSize != 100 {
		t.Errorf("max_list_size: got %d, want 100", base.MaxListSize)
	}
}

func TestMergeZeroMaxListSizePreservesBase(t *testing.T) {
	base := storage.Config{
		ContainerName:    "documents",
		ConnectionString: "base-conn",
		MaxListSize:      50,
	}

	overlay := storage.Config{}
	base.Merge(&overlay)

	if base.MaxListSize != 50 {
		t.Errorf("max_list_size: got %d, want 50 (preserved)", base.MaxListSize)
	}
}
