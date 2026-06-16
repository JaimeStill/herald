package cli

import (
	"context"
	"flag"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"

	"github.com/JaimeStill/herald/internal/documents"
	"github.com/JaimeStill/herald/pkg/pagination"
)

// UploadDocument uploads a single file with its external-system linkage and
// returns the created Document.
func (c *Client) UploadDocument(ctx context.Context, file string, externalID int, platform string) (*documents.Document, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", file, err)
	}

	fields := map[string]string{
		"external_id":       strconv.Itoa(externalID),
		"external_platform": platform,
	}

	var doc documents.Document
	if err := c.postMultipart(ctx, "/api/documents", fields, "file", filepath.Base(file), data, &doc); err != nil {
		return nil, err
	}
	return &doc, nil
}

// ListDocuments returns one page of documents matching the query parameters.
func (c *Client) ListDocuments(ctx context.Context, query url.Values) (*pagination.PageResult[documents.Document], error) {
	var result pagination.PageResult[documents.Document]
	if err := c.getJSON(ctx, "/api/documents", query, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetDocument returns a single document by ID.
func (c *Client) GetDocument(ctx context.Context, id string) (*documents.Document, error) {
	var doc documents.Document
	if err := c.getJSON(ctx, "/api/documents/"+id, nil, &doc); err != nil {
		return nil, err
	}
	return &doc, nil
}

// documentsUpload uploads one file with its external-system linkage.
func documentsUpload(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("documents upload", flag.ContinueOnError)
	flags := bindFlags(fs)
	file := fs.String("file", "", "file to upload (required)")
	externalID := fs.Int("external-id", 0, "external id (required)")
	platform := fs.String("platform", "", "external platform (required)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if *file == "" {
		return fmt.Errorf("--file is required")
	}
	if *platform == "" {
		return fmt.Errorf("--platform is required")
	}
	if !flagProvided(fs, "external-id") {
		return fmt.Errorf("--external-id is required (it is the program-of-record join key)")
	}

	s, err := Load(flags)
	if err != nil {
		return err
	}
	client, err := NewClient(s)
	if err != nil {
		return err
	}

	doc, err := client.UploadDocument(ctx, *file, *externalID, *platform)
	if err != nil {
		return err
	}
	return emit(s.Output, doc)
}

// documentsList queries one page of documents with optional filters.
func documentsList(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("documents list", flag.ContinueOnError)
	flags := bindFlags(fs)
	status := fs.String("status", "", "filter by status (pending|review|complete)")
	platform := fs.String("platform", "", "filter by external platform")
	externalID := fs.String("external-id", "", "filter by external id")
	classification := fs.String("classification", "", "filter by classification")
	confidence := fs.String("confidence", "", "filter by confidence")
	contentType := fs.String("content-type", "", "filter by content type")
	filename := fs.String("filename", "", "filter by filename (contains)")
	search := fs.String("search", "", "search term")
	sort := fs.String("sort", "", "sort fields, e.g. -uploaded_at")
	page := fs.Int("page", 0, "page number")
	pageSize := fs.Int("page-size", 0, "page size")
	if err := fs.Parse(args); err != nil {
		return err
	}

	s, err := Load(flags)
	if err != nil {
		return err
	}
	client, err := NewClient(s)
	if err != nil {
		return err
	}

	query := url.Values{}
	setQuery(query, "status", *status)
	setQuery(query, "external_platform", *platform)
	setQuery(query, "external_id", *externalID)
	setQuery(query, "classification", *classification)
	setQuery(query, "confidence", *confidence)
	setQuery(query, "content_type", *contentType)
	setQuery(query, "filename", *filename)
	setQuery(query, "search", *search)
	setQuery(query, "sort", *sort)
	if *page > 0 {
		query.Set("page", strconv.Itoa(*page))
	}
	if *pageSize > 0 {
		query.Set("page_size", strconv.Itoa(*pageSize))
	}

	result, err := client.ListDocuments(ctx, query)
	if err != nil {
		return err
	}
	return emit(s.Output, result)
}

// documentsGet fetches a single document by ID.
func documentsGet(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("documents get", flag.ContinueOnError)
	flags := bindFlags(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}
	id := fs.Arg(0)
	if id == "" {
		return fmt.Errorf("usage: herald documents get <id>")
	}

	s, err := Load(flags)
	if err != nil {
		return err
	}
	client, err := NewClient(s)
	if err != nil {
		return err
	}

	doc, err := client.GetDocument(ctx, id)
	if err != nil {
		return err
	}
	return emit(s.Output, doc)
}

// setQuery sets key=val on q when val is non-empty.
func setQuery(q url.Values, key, val string) {
	if val != "" {
		q.Set(key, val)
	}
}

// flagProvided reports whether fs saw the named flag on the command line.
func flagProvided(fs *flag.FlagSet, name string) bool {
	found := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}
