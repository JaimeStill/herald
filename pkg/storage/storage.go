// Package storage provides blob storage operations with an Azure Blob Storage implementation.
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

// System manages blob storage operations and lifecycle coordination.
type System interface {
	// Start registers a startup hook that initializes the storage container.
	Start(lc *lifecycle.Coordinator) error
	// Upload streams data to a blob at the given key with the specified content type.
	Upload(ctx context.Context, key string, reader io.Reader, contentType string) error
	// Download returns a stream for the blob at the given key. The caller must close the reader.
	// Returns ErrNotFound if the blob does not exist.
	Download(ctx context.Context, key string) (io.ReadCloser, error)
	// Delete removes the blob at the given key. Returns ErrNotFound if the blob does not exist.
	Delete(ctx context.Context, key string) error
	// Exists reports whether a blob exists at the given key.
	Exists(ctx context.Context, key string) (bool, error)
}

type azure struct {
	client    *azblob.Client
	container string
	logger    *slog.Logger
}

// New creates a storage system from the given configuration.
// It validates the connection string and creates the Azure client
// but does not establish a connection until Start is called.
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

	blobClient := a.client.
		ServiceClient().
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
