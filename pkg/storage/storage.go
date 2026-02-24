// Package storage provides blob storage operations with an Azure Blob Storage implementation.
package storage

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/bloberror"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"

	"github.com/JaimeStill/herald/pkg/lifecycle"
)

// MaxListCap is the maximum number of blobs that can be returned in a single list request.
// This matches Azure Blob Storage's server-side ceiling.
const MaxListCap int32 = 5000

// BlobMeta contains metadata about a single blob in storage.
type BlobMeta struct {
	Name          string    `json:"name"`
	ContentType   string    `json:"content_type"`
	ContentLength int64     `json:"content_length"`
	LastModified  time.Time `json:"last_modified"`
	ETag          string    `json:"etag"`
	CreatedAt     time.Time `json:"created_at"`
}

// BlobList holds a page of blob metadata with an optional continuation marker
// for marker-based pagination.
type BlobList struct {
	Blobs      []BlobMeta `json:"blobs"`
	NextMarker string     `json:"next_marker,omitempty"`
}

// BlobResult bundles blob metadata with the download body stream.
// The caller must close Body when finished reading.
type BlobResult struct {
	BlobMeta
	Body io.ReadCloser `json:"-"`
}

// System manages blob storage operations and lifecycle coordination.
type System interface {
	// Start registers a startup hook that initializes the storage container.
	Start(lc *lifecycle.Coordinator) error

	// List returns a page of blob metadata filtered by prefix.
	// Marker is an opaque continuation token from a previous BlobList.
	List(ctx context.Context, prefix string, marker string, maxResults int32) (*BlobList, error)
	// Find returns metadata for a single blob by key.
	// Returns ErrNotFound if the blob does not exist.
	Find(ctx context.Context, key string) (*BlobMeta, error)

	// Upload streams data to a blob at the given key with the specified content type.
	Upload(ctx context.Context, key string, reader io.Reader, contentType string) error
	// Download returns a stream for the blob at the given key. The caller must close the reader.
	// Returns ErrNotFound if the blob does not exist.
	Download(ctx context.Context, key string) (*BlobResult, error)
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

func (a *azure) List(
	ctx context.Context,
	prefix string,
	marker string,
	maxResults int32,
) (*BlobList, error) {
	containerClient := a.client.ServiceClient().NewContainerClient(a.container)

	opts := &container.ListBlobsFlatOptions{
		MaxResults: &maxResults,
	}
	if prefix != "" {
		opts.Prefix = &prefix
	}
	if marker != "" {
		opts.Marker = &marker
	}

	pager := containerClient.NewListBlobsFlatPager(opts)
	if !pager.More() {
		return &BlobList{Blobs: []BlobMeta{}}, nil
	}

	resp, err := pager.NextPage(ctx)
	if err != nil {
		return nil, fmt.Errorf("list blobs: %w", err)
	}

	blobs := make([]BlobMeta, 0, len(resp.Segment.BlobItems))
	for _, b := range resp.Segment.BlobItems {
		var meta BlobMeta
		if b.Name != nil {
			meta.Name = *b.Name
		}
		if b.Properties != nil {
			if b.Properties.ContentType != nil {
				meta.ContentType = *b.Properties.ContentType
			}
			if b.Properties.ContentLength != nil {
				meta.ContentLength = *b.Properties.ContentLength
			}
			if b.Properties.LastModified != nil {
				meta.LastModified = *b.Properties.LastModified
			}
			if b.Properties.ETag != nil {
				meta.ETag = string(*b.Properties.ETag)
			}
			if b.Properties.CreationTime != nil {
				meta.CreatedAt = *b.Properties.CreationTime
			}
		}
		blobs = append(blobs, meta)
	}

	result := &BlobList{Blobs: blobs}
	if resp.NextMarker != nil && *resp.NextMarker != "" {
		result.NextMarker = *resp.NextMarker
	}
	return result, nil
}

func (a *azure) Find(ctx context.Context, key string) (*BlobMeta, error) {
	if err := validateKey(key); err != nil {
		return nil, err
	}

	blobClient := a.client.
		ServiceClient().
		NewContainerClient(a.container).
		NewBlobClient(key)

	resp, err := blobClient.GetProperties(ctx, nil)
	if err != nil {
		if bloberror.HasCode(err, bloberror.BlobNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get blob properties %s: %w", key, err)
	}

	meta := &BlobMeta{Name: key}
	if resp.ContentType != nil {
		meta.ContentType = *resp.ContentType
	}
	if resp.ContentLength != nil {
		meta.ContentLength = *resp.ContentLength
	}
	if resp.LastModified != nil {
		meta.LastModified = *resp.LastModified
	}
	if resp.ETag != nil {
		meta.ETag = string(*resp.ETag)
	}
	if resp.CreationTime != nil {
		meta.CreatedAt = *resp.CreationTime
	}
	return meta, nil
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

func (a *azure) Download(ctx context.Context, key string) (*BlobResult, error) {
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

	result := &BlobResult{
		BlobMeta: BlobMeta{Name: key},
		Body:     resp.Body,
	}

	if resp.ContentType != nil {
		result.ContentType = *resp.ContentType
	}
	if resp.ContentLength != nil {
		result.ContentLength = *resp.ContentLength
	}

	return result, nil
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

// ParseMaxResults parses a max_results query parameter string into an int32,
// clamping the value at MaxListCap. Returns fallback when s is empty.
func ParseMaxResults(s string, fallback int32) (int32, error) {
	if s == "" {
		return fallback, nil
	}

	n, err := strconv.Atoi(s)
	if err != nil || n < 1 {
		return 0, fmt.Errorf("invalid max_results parameter")
	}

	return min(int32(n), MaxListCap), nil
}
