# Changelog

## [v0.1.2] - 2026-06-02

### Changed

- Broaden client retry policy to cover all transient server errors: retry HTTP 408, 429, and all 5xx responses (previously only 429/502/503/504), aligning with OpenAI/Anthropic SDK behavior. Transient 500s are now retried with the existing exponential backoff + jitter.

## [v0.1.1] - 2026-04-06

### Fixed

- Fix nil pointer dereference in streaming responses when format returns `(nil, nil)` for unrecognized stream events (e.g., Converse format skipping unknown event types)

## [v0.1.0] - 2026-04-06

### Added

- Agent interface with Chat, ChatStream, Vision, VisionStream, Tools, and Embed methods
- Explicit dependency injection constructor: `New(cfg, provider, format)`
- Client package with HTTP orchestration, retry logic, and health tracking
- Request package with ChatRequest, VisionRequest, ToolsRequest, and EmbeddingsRequest
- Registry package for named agent configuration with lazy instantiation
- Mock package with MockAgent, MockClient, MockProvider, and MockFormat
- AgentError type with categorization (init, llm) and contextual metadata
