package request

import (
	"github.com/tailored-agentic-units/format"
	"github.com/tailored-agentic-units/protocol"
	"github.com/tailored-agentic-units/protocol/model"
	"github.com/tailored-agentic-units/provider"
)

// EmbeddingsRequest represents an embeddings protocol request.
// Separates input text from model configuration options.
// Does not use messages array — input is the primary data field.
type EmbeddingsRequest struct {
	input   any
	options map[string]any
	prov    provider.Provider
	fmt     format.Format
	mdl     *model.Model
}

// NewEmbeddings creates a new EmbeddingsRequest with the given components.
// Input is the text to embed (string or []string for batch).
// Options specify model configuration (encoding_format, dimensions, etc.).
func NewEmbeddings(p provider.Provider, f format.Format, m *model.Model, input any, opts map[string]any) *EmbeddingsRequest {
	return &EmbeddingsRequest{
		input:   input,
		options: opts,
		prov:    p,
		fmt:     f,
		mdl:     m,
	}
}

// Protocol returns the Embeddings protocol identifier.
func (r *EmbeddingsRequest) Protocol() protocol.Protocol {
	return protocol.Embeddings
}

// Headers returns the HTTP headers for an embeddings request.
func (r *EmbeddingsRequest) Headers() map[string]string {
	return map[string]string{
		"Content-Type": "application/json",
	}
}

// Marshal delegates to the format for wire-format-specific JSON encoding.
func (r *EmbeddingsRequest) Marshal() ([]byte, error) {
	return r.fmt.Marshal(protocol.Embeddings, &format.EmbeddingsData{
		Model:   r.mdl.Name,
		Input:   r.input,
		Options: r.options,
	})
}

// Provider returns the provider for this request.
func (r *EmbeddingsRequest) Provider() provider.Provider {
	return r.prov
}

// Model returns the model for this request.
func (r *EmbeddingsRequest) Model() *model.Model {
	return r.mdl
}

// Format returns the wire format for response parsing.
func (r *EmbeddingsRequest) Format() format.Format {
	return r.fmt
}
