# Changelog

## [v0.1.0] - 2026-04-06

Initial release. Wire format abstraction for LLM APIs.

**Added**:
- `Format` interface with `Marshal`, `Parse`, `ParseStreamChunk` methods
- Data types: `ChatData`, `VisionData`, `ToolsData`, `EmbeddingsData`, `ToolDefinition`, `Image`
- Thread-safe format registry with `Register`, `Create`, `ListFormats`
- `Factory` function type for format constructors
- Explicit registration pattern (no `init()` auto-registration)
