# Changelog

## [v0.1.0] - 2026-04-06

Initial release. Ollama provider for the TAU ecosystem.

**Added**:
- `OllamaProvider` implementing `provider.Provider` for Ollama services
- OpenAI-compatible API with automatic `/v1` suffix
- Support for Chat, Vision, Tools, and Embeddings protocols
- Optional bearer token and API key authentication
- SSE streaming via `provider/streaming.SSEReader`
- `Register()` for explicit provider registration
- `NewOllama` factory constructor
