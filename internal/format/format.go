// Package format defines Herald's document-format abstraction: a Handler
// interface that knows how to extract and re-render pages for a given MIME
// type, and a Registry that dispatches by content type. Consumers depend on
// the abstraction (documents handler for upload validation, workflow nodes
// for extraction and enhancement), while concrete handlers (PDF, image)
// live alongside and are composed at startup.
package format

import (
	"context"
	"fmt"
	"io"
	"slices"

	"github.com/JaimeStill/herald/internal/state"
)

// Handler processes a single document format. Implementations know how to
// download a document from a SourceReader and render its pages to PNG files
// on disk, and how to re-render a specific page with enhancement filters
// applied. Handlers are stateless; all request-scoped data flows through
// method arguments.
type Handler interface {
	// ID returns a short stable identifier for the format (e.g. "pdf", "image").
	// Used in logs and event payloads; not intended for end-user display.
	ID() string

	// ContentTypes returns the MIME types this handler accepts. The Registry
	// indexes handlers by these values for Lookup dispatch.
	ContentTypes() []string

	// Extract downloads the document from src, writes one or more PNG images
	// to tempDir, and returns per-page ClassificationPage entries. For
	// single-image formats len(pages) == 1. The caller owns tempDir and is
	// responsible for its cleanup.
	Extract(
		ctx context.Context,
		src SourceReader,
		tempDir string,
	) ([]state.ClassificationPage, error)

	// Enhance re-renders a specific page with the given enhancement filters
	// and returns the new image path. The handler decides whether to render
	// from the original source (PDFs) or from a previously-extracted
	// intermediate (images). tempDir must already contain the prior-pass
	// artifacts from Extract.
	Enhance(
		ctx context.Context,
		tempDir string,
		page *state.ClassificationPage,
		settings *state.EnhanceSettings,
	) (string, error)
}

// SourceReader abstracts "download this document" so handlers do not depend
// on the storage subsystem directly. The workflow's init node wraps blob
// storage via an adapter; tests substitute a file-backed implementation.
// Open is expected to return a fresh reader on each call, though typical
// handler usage reads once and copies bytes eagerly.
type SourceReader interface {
	Open(ctx context.Context) (io.ReadCloser, error)
	ContentType() string
	Filename() string
}

// Registry maps MIME content types to the Handler that services them. The
// handlers map is the single source of truth; SupportedContentTypes derives
// a deterministic list by sorting its keys.
type Registry struct {
	handlers map[string]Handler
}

// NewRegistry composes a Registry from the given handlers. Each handler's
// ContentTypes are indexed in the internal map; if two handlers declare the
// same content type, the later one wins.
func NewRegistry(handlers ...Handler) *Registry {
	r := &Registry{handlers: make(map[string]Handler)}
	for _, h := range handlers {
		for _, ct := range h.ContentTypes() {
			r.handlers[ct] = h
		}
	}
	return r
}

// Lookup returns the Handler registered for contentType, or an error wrapping
// ErrUnsupportedFormat when no handler matches. Callers typically use
// errors.Is(err, ErrUnsupportedFormat) to distinguish registry misses from
// other failure modes.
func (r *Registry) Lookup(contentType string) (Handler, error) {
	h, ok := r.handlers[contentType]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedFormat, contentType)
	}
	return h, nil
}

// SupportedContentTypes returns all registered MIME types in sorted order.
// The result is safe to include in user-facing error messages; ordering is
// stable across calls so tests and logs stay deterministic.
func (r *Registry) SupportedContentTypes() []string {
	out := make([]string, 0, len(r.handlers))
	for ct := range r.handlers {
		out = append(out, ct)
	}
	slices.Sort(out)
	return out
}

// readAll is a shared helper that fully buffers a SourceReader into memory.
// Handlers copy bytes eagerly, so single-use SourceReader implementations
// are fine; no pooling is needed.
func readAll(ctx context.Context, src SourceReader) ([]byte, error) {
	rc, err := src.Open(ctx)
	if err != nil {
		return nil, fmt.Errorf("open source: %w", err)
	}
	defer rc.Close()
	data, err := io.ReadAll(rc)
	if err != nil {
		return nil, fmt.Errorf("read source: %w", err)
	}
	return data, nil
}
