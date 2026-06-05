# Changelog

## [v0.1.0] - 2026-04-06

Initial release. Foundation types for the TAU ecosystem.

**Added**:
- `Protocol` type with constants: Chat, Vision, Tools, Embeddings, Audio
- `Message` type with typed `Role`, `ToolCalls`, and `ToolCallID` fields
- `Role` type with constants: RoleSystem, RoleUser, RoleAssistant, RoleTool
- `ToolCall` and `ToolFunction` types for function calling
- Convenience constructors: `UserMessage()`, `SystemMessage()`, `AssistantMessage()`, `ToolMessage()`
- `config` package: `AgentConfig`, `ClientConfig`, `ProviderConfig`, `ModelConfig` with JSON serialization, defaults, and merge support
- `AgentConfig.Format` field for wire format selection (defaults to "openai")
- `response` package: Unified `Response` type with `ContentBlock` interface (`TextBlock`, `ToolUseBlock`)
- `StreamingResponse` type for streaming model responses
- `EmbeddingsResponse` type for vector outputs
- `TokenUsage` type for token consumption tracking
- `model` package: Runtime `Model` type bridging config to domain
- `streaming` package: `StreamReader` interface and `StreamLine` type for streamed responses
