// Package documents implements the document domain for Herald.
// It provides types, data access, and business logic for document
// upload, registration, metadata management, and blob storage integration.
package documents

import (
	"time"

	"github.com/google/uuid"
)

// Document represents a registered document with its metadata and blob storage reference.
type Document struct {
	ID               uuid.UUID `json:"id"`
	ExternalID       int       `json:"external_id"`
	ExternalPlatform string    `json:"external_platform"`
	Filename         string    `json:"filename"`
	ContentType      string    `json:"content_type"`
	SizeBytes        int64     `json:"size_bytes"`
	PageCount        *int      `json:"page_count"`
	StorageKey       string    `json:"storage_key"`
	Status           string    `json:"status"`
	UploadedAt       time.Time `json:"uploaded_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// CreateCommand carries the data needed to upload and register a new document.
// Data holds the raw file bytes. PageCount is optional and may be extracted
// by the caller via pdfcpu; nil values are stored as NULL.
type CreateCommand struct {
	Data             []byte
	Filename         string
	ContentType      string
	ExternalID       int
	ExternalPlatform string
	PageCount        *int
}

// BatchResult reports the outcome of a single file within a batch upload.
// On success, Document is populated and Error is empty.
// On failure, Error describes the problem and Document is nil.
type BatchResult struct {
	Document *Document `json:"document,omitempty"`
	Filename string    `json:"filename"`
	Error    string    `json:"error,omitempty"`
}
