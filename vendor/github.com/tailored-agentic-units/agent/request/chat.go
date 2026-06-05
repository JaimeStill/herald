package request

import (
	"github.com/tailored-agentic-units/format"
	"github.com/tailored-agentic-units/protocol"
	"github.com/tailored-agentic-units/protocol/model"
	"github.com/tailored-agentic-units/provider"
)

// ChatRequest represents a chat protocol request.
// Encapsulates conversation messages, model configuration options,
// and the provider/format/model needed for execution.
type ChatRequest struct {
	messages []protocol.Message
	options  map[string]any
	prov     provider.Provider
	fmt      format.Format
	mdl      *model.Model
}

// NewChat creates a new ChatRequest with the given components.
// Messages contain the conversation history.
// Options specify model configuration (temperature, max_tokens, etc.).
func NewChat(p provider.Provider, f format.Format, m *model.Model, messages []protocol.Message, opts map[string]any) *ChatRequest {
	return &ChatRequest{
		messages: messages,
		options:  opts,
		prov:     p,
		fmt:      f,
		mdl:      m,
	}
}

// Protocol returns the Chat protocol identifier.
func (r *ChatRequest) Protocol() protocol.Protocol {
	return protocol.Chat
}

// Headers returns the HTTP headers for a chat request.
func (r *ChatRequest) Headers() map[string]string {
	return map[string]string{
		"Content-Type": "application/json",
	}
}

// Marshal delegates to the format for wire-format-specific JSON encoding.
func (r *ChatRequest) Marshal() ([]byte, error) {
	return r.fmt.Marshal(protocol.Chat, &format.ChatData{
		Model:    r.mdl.Name,
		Messages: r.messages,
		Options:  r.options,
	})
}

// Provider returns the provider for this request.
func (r *ChatRequest) Provider() provider.Provider {
	return r.prov
}

// Model returns the model for this request.
func (r *ChatRequest) Model() *model.Model {
	return r.mdl
}

// Format returns the wire format for response parsing.
func (r *ChatRequest) Format() format.Format {
	return r.fmt
}
