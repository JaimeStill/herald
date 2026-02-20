package storage_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/JaimeStill/herald/pkg/storage"
)

const azuriteConnString = "DefaultEndpointsProtocol=http;AccountName=heraldstore;AccountKey=Eby8vdM02xNOcqFlqUwJPLlmEtlCDXJ1OUzFT50uSRZ6IFsuFq2UVErCz4I6tq/K1SZFPTOtr/KBHBeksoGMGw==;BlobEndpoint=http://127.0.0.1:10000/heraldstore;"

func TestNewReturnsSystem(t *testing.T) {
	cfg := &storage.Config{
		ContainerName:    "documents",
		ConnectionString: azuriteConnString,
	}

	sys, err := storage.New(cfg, slog.Default())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if sys == nil {
		t.Fatal("New() returned nil system")
	}
}

func TestNewInvalidConnectionString(t *testing.T) {
	cfg := &storage.Config{
		ContainerName:    "documents",
		ConnectionString: "not-a-connection-string",
	}

	_, err := storage.New(cfg, slog.Default())
	if err == nil {
		t.Fatal("expected error for invalid connection string, got nil")
	}
}

func TestSentinelErrors(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		wantMsg string
	}{
		{
			name:    "ErrNotFound",
			err:     storage.ErrNotFound,
			wantMsg: "blob not found",
		},
		{
			name:    "ErrEmptyKey",
			err:     storage.ErrEmptyKey,
			wantMsg: "storage key must not be empty",
		},
		{
			name:    "ErrInvalidKey",
			err:     storage.ErrInvalidKey,
			wantMsg: "storage key contains invalid path segment",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !errors.Is(tt.err, tt.err) {
				t.Errorf("%s should match itself", tt.name)
			}
			if tt.err.Error() != tt.wantMsg {
				t.Errorf("%s.Error() = %q, want %q", tt.name, tt.err.Error(), tt.wantMsg)
			}
		})
	}
}

func TestKeyValidation(t *testing.T) {
	cfg := &storage.Config{
		ContainerName:    "documents",
		ConnectionString: azuriteConnString,
	}

	sys, err := storage.New(cfg, slog.Default())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	tests := []struct {
		name    string
		key     string
		wantErr error
	}{
		{
			name:    "empty key",
			key:     "",
			wantErr: storage.ErrEmptyKey,
		},
		{
			name:    "path traversal",
			key:     "documents/../secrets/key",
			wantErr: storage.ErrInvalidKey,
		},
		{
			name:    "double dot in middle",
			key:     "docs/..hidden/file.pdf",
			wantErr: storage.ErrInvalidKey,
		},
	}

	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := sys.Upload(ctx, tt.key, bytes.NewReader(nil), "application/pdf")
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Upload() error = %v, want %v", err, tt.wantErr)
			}

			_, err = sys.Download(ctx, tt.key)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Download() error = %v, want %v", err, tt.wantErr)
			}

			err = sys.Delete(ctx, tt.key)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Delete() error = %v, want %v", err, tt.wantErr)
			}

			_, err = sys.Exists(ctx, tt.key)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Exists() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}
