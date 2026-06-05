// Package format defines the wire format abstraction for marshaling and parsing
// LLM provider requests and responses across different API formats.
//
// The root module exposes the Format interface, data types, and a registry.
// Concrete implementations (OpenAI, Converse) live in sub-modules with their
// own go.mod files, isolating provider-specific dependencies.
package format
