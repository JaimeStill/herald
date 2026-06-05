// Package protocol provides the foundation types for LLM interaction in the TAU ecosystem.
//
// It defines the Protocol type representing different LLM capabilities, the Message type
// for conversation structures, and supporting types for configuration, response handling,
// model runtime, and streaming.
//
// Sub-packages:
//   - config: Agent, client, provider, and model configuration with JSON serialization
//   - response: Unified response types with content blocks (text, tool use)
//   - model: Runtime model type bridging configuration to domain
//   - streaming: StreamReader interface for streamed LLM responses
package protocol
