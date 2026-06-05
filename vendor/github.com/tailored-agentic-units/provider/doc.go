// Package provider defines the Provider interface for LLM service transport,
// authentication, and endpoint routing.
//
// The root module exposes the Provider interface, BaseProvider, Request type,
// and a thread-safe registry. Concrete implementations (Ollama, Azure, Bedrock)
// live in sub-modules with their own go.mod files, isolating cloud-specific
// SDK dependencies from consumers that only need the interface.
package provider
