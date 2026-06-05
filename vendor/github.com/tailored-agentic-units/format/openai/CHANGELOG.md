# Changelog

## [v0.1.0] - 2026-04-06

Initial release. OpenAI-compatible wire format for the TAU ecosystem.

**Added**:
- `Format` implementing `format.Format` for OpenAI-compatible APIs
- Marshal support: Chat, Vision, Tools, Embeddings protocols
- Parse support: Chat/Vision responses, Tools responses with `ToolUseBlock`, Embeddings
- Streaming support: `ParseStreamChunk` for SSE delta events
- `Register()` for explicit format registration
- `Factory()` constructor for registry integration
