package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"

	"github.com/JaimeStill/herald/internal/classifications"
	"github.com/JaimeStill/herald/pkg/pagination"
)

// Classify triggers classification for a document and consumes the resulting
// Server-Sent Events stream to completion, returning the classification carried
// by the terminal complete event. Progress events are ignored; an error event
// surfaces as an error.
func (c *Client) Classify(ctx context.Context, documentID string) (*classifications.Classification, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.postStream(ctx, "/api/classifications/"+documentID)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return parseClassificationStream(resp.Body)
}

// ListClassifications returns one page of classifications matching the query.
func (c *Client) ListClassifications(ctx context.Context, query url.Values) (*pagination.PageResult[classifications.Classification], error) {
	var result pagination.PageResult[classifications.Classification]
	if err := c.getJSON(ctx, "/api/classifications", query, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetClassification returns a single classification by ID.
func (c *Client) GetClassification(ctx context.Context, id string) (*classifications.Classification, error) {
	var cl classifications.Classification
	if err := c.getJSON(ctx, "/api/classifications/"+id, nil, &cl); err != nil {
		return nil, err
	}
	return &cl, nil
}

// GetClassificationByDocument returns the classification for a document ID.
func (c *Client) GetClassificationByDocument(ctx context.Context, documentID string) (*classifications.Classification, error) {
	var cl classifications.Classification
	if err := c.getJSON(ctx, "/api/classifications/document/"+documentID, nil, &cl); err != nil {
		return nil, err
	}
	return &cl, nil
}

// parseClassificationStream reads an SSE stream and resolves on the terminal
// event. Each data line carries the full ExecutionEvent envelope
// ({type, timestamp, data}); a complete event yields the classification from its
// nested data, an error event yields an error. The stream ending without either
// is itself an error.
func parseClassificationStream(r io.Reader) (*classifications.Classification, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" {
			continue
		}

		var env struct {
			Type string          `json:"type"`
			Data json.RawMessage `json:"data"`
		}
		if err := json.Unmarshal([]byte(data), &env); err != nil {
			return nil, fmt.Errorf("decode stream event: %w", err)
		}

		switch env.Type {
		case "complete":
			var cl classifications.Classification
			if err := json.Unmarshal(env.Data, &cl); err != nil {
				return nil, fmt.Errorf("decode classification: %w", err)
			}
			return &cl, nil
		case "error":
			return nil, classificationStreamError(env.Data)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read classification stream: %w", err)
	}
	return nil, fmt.Errorf("classification stream ended without a result")
}

// classificationStreamError extracts the message from an error event's data
// payload ({"message": ..., "node": ...}).
func classificationStreamError(data json.RawMessage) error {
	var ed struct {
		Message string `json:"message"`
	}
	if json.Unmarshal(data, &ed) == nil && ed.Message != "" {
		return fmt.Errorf("classification failed: %s", ed.Message)
	}
	return fmt.Errorf("classification failed")
}

// classify triggers classification for a single document and emits the result.
func classify(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("classify", flag.ContinueOnError)
	flags := bindFlags(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}
	id := fs.Arg(0)
	if id == "" {
		return fmt.Errorf("usage: herald classify <document-id>")
	}

	s, err := Load(flags)
	if err != nil {
		return err
	}
	client, err := NewClient(s)
	if err != nil {
		return err
	}

	cl, err := client.Classify(ctx, id)
	if err != nil {
		return err
	}
	return emit(s.Output, cl)
}

// classificationsList queries one page of classifications with optional filters.
func classificationsList(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("classifications list", flag.ContinueOnError)
	flags := bindFlags(fs)
	classification := fs.String("classification", "", "filter by classification")
	confidence := fs.String("confidence", "", "filter by confidence (HIGH|MEDIUM|LOW)")
	documentID := fs.String("document-id", "", "filter by document id")
	validatedBy := fs.String("validated-by", "", "filter by validator")
	search := fs.String("search", "", "search term")
	sort := fs.String("sort", "", "sort fields, e.g. -classified_at")
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
	setQuery(query, "classification", *classification)
	setQuery(query, "confidence", *confidence)
	setQuery(query, "document_id", *documentID)
	setQuery(query, "validated_by", *validatedBy)
	setQuery(query, "search", *search)
	setQuery(query, "sort", *sort)
	if *page > 0 {
		query.Set("page", strconv.Itoa(*page))
	}
	if *pageSize > 0 {
		query.Set("page_size", strconv.Itoa(*pageSize))
	}

	result, err := client.ListClassifications(ctx, query)
	if err != nil {
		return err
	}
	return emit(s.Output, result)
}

// classificationsGet fetches a single classification by ID.
func classificationsGet(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("classifications get", flag.ContinueOnError)
	flags := bindFlags(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}
	id := fs.Arg(0)
	if id == "" {
		return fmt.Errorf("usage: herald classifications get <id>")
	}

	s, err := Load(flags)
	if err != nil {
		return err
	}
	client, err := NewClient(s)
	if err != nil {
		return err
	}

	cl, err := client.GetClassification(ctx, id)
	if err != nil {
		return err
	}
	return emit(s.Output, cl)
}

// classificationsByDocument fetches the classification for a document ID.
func classificationsByDocument(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("classifications by-document", flag.ContinueOnError)
	flags := bindFlags(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}
	id := fs.Arg(0)
	if id == "" {
		return fmt.Errorf("usage: herald classifications by-document <document-id>")
	}

	s, err := Load(flags)
	if err != nil {
		return err
	}
	client, err := NewClient(s)
	if err != nil {
		return err
	}

	cl, err := client.GetClassificationByDocument(ctx, id)
	if err != nil {
		return err
	}
	return emit(s.Output, cl)
}
