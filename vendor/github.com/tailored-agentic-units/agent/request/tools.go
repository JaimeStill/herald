package request

import (
	"github.com/tailored-agentic-units/format"
	"github.com/tailored-agentic-units/protocol"
	"github.com/tailored-agentic-units/protocol/model"
	"github.com/tailored-agentic-units/provider"
)

// ToolsRequest represents a tools (function calling) protocol request.
// Separates tool definitions from model configuration options.
type ToolsRequest struct {
	messages []protocol.Message
	tools    []format.ToolDefinition
	options  map[string]any
	prov     provider.Provider
	fmt      format.Format
	mdl      *model.Model
}

// NewTools creates a new ToolsRequest with the given components.
// Messages contain the conversation history.
// Tools define the available functions the model can call.
// Options specify model configuration (temperature, max_tokens, etc.).
func NewTools(p provider.Provider, f format.Format, m *model.Model, messages []protocol.Message, tools []format.ToolDefinition, opts map[string]any) *ToolsRequest {
	return &ToolsRequest{
		messages: messages,
		tools:    tools,
		options:  opts,
		prov:     p,
		fmt:      f,
		mdl:      m,
	}
}

// Protocol returns the Tools protocol identifier.
func (r *ToolsRequest) Protocol() protocol.Protocol {
	return protocol.Tools
}

// Headers returns the HTTP headers for a tools request.
func (r *ToolsRequest) Headers() map[string]string {
	return map[string]string{
		"Content-Type": "application/json",
	}
}

// Marshal delegates to the format for wire-format-specific JSON encoding.
func (r *ToolsRequest) Marshal() ([]byte, error) {
	return r.fmt.Marshal(protocol.Tools, &format.ToolsData{
		Model:    r.mdl.Name,
		Messages: r.messages,
		Tools:    r.tools,
		Options:  r.options,
	})
}

// Provider returns the provider for this request.
func (r *ToolsRequest) Provider() provider.Provider {
	return r.prov
}

// Model returns the model for this request.
func (r *ToolsRequest) Model() *model.Model {
	return r.mdl
}

// Format returns the wire format for response parsing.
func (r *ToolsRequest) Format() format.Format {
	return r.fmt
}
