// Package model provides the Model type representing a configured LLM model at runtime.
// It stores the model name and protocol-specific default options,
// bridging JSON configuration with runtime domain types.
package model

import (
	"github.com/tailored-agentic-units/protocol"
	"github.com/tailored-agentic-units/protocol/config"
)

// Model represents a configured LLM model at runtime.
// It stores the model name and protocol-specific default options.
// This is the domain type used during execution, separate from JSON configuration.
type Model struct {
	// Name is the model identifier (e.g., "gpt-4o", "claude-3-opus", "llama3.1:8b")
	Name string

	// Options holds protocol-specific default options.
	// Keys are protocols (Chat, Vision, Tools, Embeddings).
	// Values are option maps for that protocol (temperature, max_tokens, etc.)
	Options map[protocol.Protocol]map[string]any
}

// New creates a Model from a ModelConfig.
// Handles conversion from string-keyed configuration to Protocol-keyed runtime model.
func New(cfg *config.ModelConfig) *Model {
	model := &Model{
		Name:    cfg.Name,
		Options: make(map[protocol.Protocol]map[string]any),
	}

	for protocolName, options := range cfg.Capabilities {
		p := protocol.Protocol(protocolName)
		model.Options[p] = options
	}

	return model
}
