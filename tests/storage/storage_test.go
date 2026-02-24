package storage_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
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

func TestMapHTTPStatus(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		want   int
	}{
		{
			name: "ErrNotFound maps to 404",
			err:  storage.ErrNotFound,
			want: http.StatusNotFound,
		},
		{
			name: "ErrEmptyKey maps to 400",
			err:  storage.ErrEmptyKey,
			want: http.StatusBadRequest,
		},
		{
			name: "ErrInvalidKey maps to 400",
			err:  storage.ErrInvalidKey,
			want: http.StatusBadRequest,
		},
		{
			name: "wrapped ErrNotFound maps to 404",
			err:  fmt.Errorf("operation failed: %w", storage.ErrNotFound),
			want: http.StatusNotFound,
		},
		{
			name: "unknown error maps to 500",
			err:  fmt.Errorf("unexpected failure"),
			want: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := storage.MapHTTPStatus(tt.err)
			if got != tt.want {
				t.Errorf("MapHTTPStatus() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestParseMaxResults(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		fallback int32
		want     int32
		wantErr  bool
	}{
		{
			name:     "empty returns fallback",
			input:    "",
			fallback: 50,
			want:     50,
		},
		{
			name:     "valid value within cap",
			input:    "100",
			fallback: 50,
			want:     100,
		},
		{
			name:     "value exceeding cap is clamped",
			input:    "9999",
			fallback: 50,
			want:     storage.MaxListCap,
		},
		{
			name:     "value at cap returns cap",
			input:    "5000",
			fallback: 50,
			want:     storage.MaxListCap,
		},
		{
			name:     "zero is invalid",
			input:    "0",
			fallback: 50,
			wantErr:  true,
		},
		{
			name:     "negative is invalid",
			input:    "-1",
			fallback: 50,
			wantErr:  true,
		},
		{
			name:     "non-numeric is invalid",
			input:    "abc",
			fallback: 50,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := storage.ParseMaxResults(tt.input, tt.fallback)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseMaxResults(%q) expected error, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseMaxResults(%q) unexpected error: %v", tt.input, err)
			}
			if got != tt.want {
				t.Errorf("ParseMaxResults(%q) = %d, want %d", tt.input, got, tt.want)
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

			_, err = sys.Find(ctx, tt.key)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Find() error = %v, want %v", err, tt.wantErr)
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
