package format

import "github.com/tailored-agentic-units/protocol"

// ChatData holds the input data for a chat completion request.
type ChatData struct {
	Model    string
	Messages []protocol.Message
	Options  map[string]any
}

// Image represents an image attachment in a provider-neutral form,
// carrying raw bytes and format metadata rather than a pre-encoded string.
type Image struct {
	Data   []byte
	Format string
	URL    string
}

// VisionData holds the input data for a vision request, combining messages with images.
type VisionData struct {
	Model         string
	Messages      []protocol.Message
	Images        []Image
	VisionOptions map[string]any
	Options       map[string]any
}

// ToolsData holds the input data for a tool-use request.
type ToolsData struct {
	Model    string
	Messages []protocol.Message
	Tools    []ToolDefinition
	Options  map[string]any
}

// ToolDefinition describes a tool that the model may invoke during a conversation.
type ToolDefinition struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

// EmbeddingsData holds the input data for an embeddings request.
type EmbeddingsData struct {
	Model   string
	Input   any
	Options map[string]any
}
