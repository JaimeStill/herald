# Changelog

## [v0.1.0] - 2026-04-06

Initial release. LLM provider abstraction for the TAU ecosystem.

**Added**:
- `Provider` interface with `Endpoint`, `Stream`, `SetHeaders`, `PrepareRequest`, `PrepareStreamRequest`
- `BaseProvider` with `Name()` and `BaseURL()` accessors
- `Request` type for decoupled HTTP request preparation
- Thread-safe provider registry with `Register`, `Create`, `ListProviders`
- `Factory` function type for provider constructors
- `streaming.SSEReader` implementing `protocol/streaming.StreamReader`
- Explicit registration pattern (no `init()` auto-registration)
