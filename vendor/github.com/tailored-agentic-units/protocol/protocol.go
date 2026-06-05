package protocol

import "strings"

// Protocol represents the type of LLM interaction operation.
// Each protocol defines a specific capability for model interaction.
type Protocol string

const (
	// Chat represents standard text-based conversation protocol.
	Chat Protocol = "chat"

	// Vision represents image understanding with multimodal inputs.
	Vision Protocol = "vision"

	// Tools represents function calling and tool execution protocol.
	Tools Protocol = "tools"

	// Embeddings represents text vectorization for semantic search.
	Embeddings Protocol = "embeddings"

	// Audio represents speech-to-text transcription protocol.
	Audio Protocol = "audio"
)

// IsValid checks if a protocol string is valid.
func IsValid(p string) bool {
	switch Protocol(p) {
	case Chat, Vision, Tools, Embeddings, Audio:
		return true
	default:
		return false
	}
}

// ValidProtocols returns a slice of all supported protocol values.
func ValidProtocols() []Protocol {
	return []Protocol{
		Chat,
		Vision,
		Tools,
		Embeddings,
		Audio,
	}
}

// ProtocolStrings returns a comma-separated string of all valid protocols.
func ProtocolStrings() string {
	valid := ValidProtocols()
	strs := make([]string, len(valid))
	for i, p := range valid {
		strs[i] = string(p)
	}
	return strings.Join(strs, ", ")
}

// SupportsStreaming returns true if the protocol supports streaming responses.
// Currently Chat, Vision, and Tools support streaming.
func (p Protocol) SupportsStreaming() bool {
	switch p {
	case Chat, Vision, Tools:
		return true
	case Embeddings, Audio:
		return false
	default:
		return false
	}
}
