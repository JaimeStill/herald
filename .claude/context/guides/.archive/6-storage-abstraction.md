# 6 - Storage Abstraction

## Problem Context

Herald needs a blob storage layer for persisting PDF documents in Azure Blob Storage. The configuration layer (`pkg/storage/config.go`) and its integration in `internal/config/config.go` already exist. This task adds the `System` interface and Azure Blob Storage implementation, completing the storage package so it can be consumed by `internal/infrastructure/` in issue #7.

## Architecture Approach

Follow the same structural pattern as `pkg/database/database.go`: a `System` interface with a private implementing struct, a `New` constructor that creates the client during cold start (no network calls), and `Start` lifecycle hooks that verify connectivity and initialize the container. All methods use streaming I/O (`io.Reader`/`io.ReadCloser`) rather than byte slices, since documents can be large PDFs.

The Azure `azblob.Client` top-level type provides convenience methods (`UploadStream`, `DownloadStream`, `DeleteBlob`, `CreateContainer`) that accept container name and blob name as parameters, eliminating the need to cache separate container/blob clients. The only exception is `Exists`, which requires descending through the service → container → blob client hierarchy to call `GetProperties`.

## Implementation

### Step 1: Add Azure Blob Storage SDK dependency

```bash
go get github.com/Azure/azure-sdk-for-go/sdk/storage/azblob
```

### Step 2: Create `pkg/storage/errors.go`

New file with sentinel errors following the `pkg/database/errors.go` pattern.

```go
package storage

import "errors"

var (
	ErrNotFound   = errors.New("blob not found")
	ErrEmptyKey   = errors.New("storage key must not be empty")
	ErrInvalidKey = errors.New("storage key contains invalid path segment")
)
```

### Step 3: Create `pkg/storage/storage.go`

New file containing the interface, private struct, constructor, lifecycle hooks, all domain methods, and key validation helper. Mirrors the single-file approach of `database.go`.

```go
package storage

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/bloberror"

	"github.com/JaimeStill/herald/pkg/lifecycle"
)

type System interface {
	Start(lc *lifecycle.Coordinator) error
	Upload(ctx context.Context, key string, reader io.Reader, contentType string) error
	Download(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
}

type azure struct {
	client    *azblob.Client
	container string
	logger    *slog.Logger
}

func New(cfg *Config, logger *slog.Logger) (System, error) {
	client, err := azblob.NewClientFromConnectionString(cfg.ConnectionString, nil)
	if err != nil {
		return nil, fmt.Errorf("create storage client: %w", err)
	}

	return &azure{
		client:    client,
		container: cfg.ContainerName,
		logger:    logger.With("system", "storage"),
	}, nil
}

func (a *azure) Start(lc *lifecycle.Coordinator) error {
	a.logger.Info("starting storage system")

	lc.OnStartup(func() {
		_, err := a.client.CreateContainer(lc.Context(), a.container, nil)
		if err != nil {
			// ContainerAlreadyExists is expected on subsequent startups
			if !bloberror.HasCode(err, bloberror.ContainerAlreadyExists) {
				a.logger.Error("storage container initialization failed", "error", err)
				return
			}
		}

		a.logger.Info("storage container ready", "container", a.container)
	})

	return nil
}

func (a *azure) Upload(ctx context.Context, key string, reader io.Reader, contentType string) error {
	if err := validateKey(key); err != nil {
		return err
	}

	opts := &azblob.UploadStreamOptions{
		HTTPHeaders: &blob.HTTPHeaders{
			BlobContentType: &contentType,
		},
	}

	_, err := a.client.UploadStream(ctx, a.container, key, reader, opts)
	if err != nil {
		return fmt.Errorf("upload blob %s: %w", key, err)
	}

	return nil
}

func (a *azure) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	if err := validateKey(key); err != nil {
		return nil, err
	}

	resp, err := a.client.DownloadStream(ctx, a.container, key, nil)
	if err != nil {
		if bloberror.HasCode(err, bloberror.BlobNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("download blob %s: %w", key, err)
	}

	return resp.Body, nil
}

func (a *azure) Delete(ctx context.Context, key string) error {
	if err := validateKey(key); err != nil {
		return err
	}

	_, err := a.client.DeleteBlob(ctx, a.container, key, nil)
	if err != nil {
		if bloberror.HasCode(err, bloberror.BlobNotFound) {
			return ErrNotFound
		}
		return fmt.Errorf("delete blob %s: %w", key, err)
	}

	return nil
}

func (a *azure) Exists(ctx context.Context, key string) (bool, error) {
	if err := validateKey(key); err != nil {
		return false, err
	}

	blobClient := a.client.ServiceClient().
		NewContainerClient(a.container).
		NewBlobClient(key)

	_, err := blobClient.GetProperties(ctx, nil)
	if err != nil {
		if bloberror.HasCode(err, bloberror.BlobNotFound) {
			return false, nil
		}
		return false, fmt.Errorf("check blob existence %s: %w", key, err)
	}

	return true, nil
}

func validateKey(key string) error {
	if key == "" {
		return ErrEmptyKey
	}
	if strings.Contains(key, "..") {
		return ErrInvalidKey
	}
	return nil
}
```

### Step 4: Run `go mod tidy`

```bash
go mod tidy
```

### Step 5: Verify build

```bash
go build ./...
go vet ./...
```

## Remediation

### R1: Azurite API version incompatibility with azblob SDK v1.6.4

The azblob SDK v1.6.4 sends API version `2026-02-06` which the current Azurite image does not support, causing all blob operations to fail with `InvalidHeaderValue`. Fix: add `--skipApiVersionCheck` to the Azurite command in `compose/azurite.yml`. This flag is recommended by Azurite itself for forward-compatibility with newer SDK versions. The container must be recreated after this change.

## Validation Criteria

- [ ] `System` interface defined with Upload, Download, Delete, Exists, Start methods
- [ ] Azure Blob Storage implementation created from connection string config
- [ ] Container initialization creates container on first startup (idempotent)
- [ ] Key validation rejects empty keys and path traversal
- [ ] Error mapping: Azure `BlobNotFound` → `ErrNotFound` sentinel
- [ ] `go build ./...` passes
- [ ] `go vet ./...` passes
- [ ] `go mod tidy` produces no changes after initial run
