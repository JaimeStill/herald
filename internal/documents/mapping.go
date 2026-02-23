package documents

import (
	"net/url"
	"strconv"

	"github.com/JaimeStill/herald/pkg/query"
	"github.com/JaimeStill/herald/pkg/repository"
)

var projection = query.
	NewProjectionMap("public", "documents", "d").
	Project("id", "ID").
	Project("external_id", "ExternalID").
	Project("external_platform", "ExternalPlatform").
	Project("filename", "Filename").
	Project("content_type", "ContentType").
	Project("size_bytes", "SizeBytes").
	Project("page_count", "PageCount").
	Project("storage_key", "StorageKey").
	Project("status", "Status").
	Project("uploaded_at", "UploadedAt").
	Project("updated_at", "UpdatedAt")

var defaultSort = query.SortField{
	Field:      "UploadedAt",
	Descending: true,
}

// Filters contains optional filtering criteria for document queries.
// Nil fields are ignored. Status, ExternalID, ExternalPlatform, and ContentType
// use exact matching. Filename and StorageKey use case-insensitive contains matching.
type Filters struct {
	Status           *string `json:"status,omitempty"`
	Filename         *string `json:"filename,omitempty"`
	ExternalID       *int    `json:"external_id,omitempty"`
	ExternalPlatform *string `json:"external_platform,omitempty"`
	ContentType      *string `json:"content_type,omitempty"`
	StorageKey       *string `json:"storage_key,omitempty"`
}

// Apply adds filter conditions to a query builder.
func (f Filters) Apply(b *query.Builder) *query.Builder {
	return b.
		WhereEquals("Status", f.Status).
		WhereContains("Filename", f.Filename).
		WhereEquals("ExternalID", f.ExternalID).
		WhereEquals("ExternalPlatform", f.ExternalPlatform).
		WhereEquals("ContentType", f.ContentType).
		WhereContains("StorageKey", f.StorageKey)
}

// FiltersFromQuery extracts filter values from URL query parameters.
func FiltersFromQuery(values url.Values) Filters {
	var f Filters

	if s := values.Get("status"); s != "" {
		f.Status = &s
	}

	if fn := values.Get("filename"); fn != "" {
		f.Filename = &fn
	}

	if eid := values.Get("external_id"); eid != "" {
		if v, err := strconv.Atoi(eid); err == nil {
			f.ExternalID = &v
		}
	}

	if ep := values.Get("external_platform"); ep != "" {
		f.ExternalPlatform = &ep
	}

	if ct := values.Get("content_type"); ct != "" {
		f.ContentType = &ct
	}

	if sk := values.Get("storage_key"); sk != "" {
		f.StorageKey = &sk
	}

	return f
}

func scanDocument(s repository.Scanner) (Document, error) {
	var d Document
	err := s.Scan(
		&d.ID,
		&d.ExternalID,
		&d.ExternalPlatform,
		&d.Filename,
		&d.ContentType,
		&d.SizeBytes,
		&d.PageCount,
		&d.StorageKey,
		&d.Status,
		&d.UploadedAt,
		&d.UpdatedAt,
	)
	return d, err
}
