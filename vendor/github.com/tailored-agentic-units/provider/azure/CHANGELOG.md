# Changelog

## [v0.1.0] - 2026-04-06

Initial release. Azure OpenAI provider for the TAU ecosystem.

**Added**:
- `AzureProvider` implementing `provider.Provider` for Azure OpenAI Service
- Deployment-based endpoint routing with API version query parameter
- API key, bearer token, and managed identity authentication
- `AzureTokenSource` for managed identity token acquisition
- SSE streaming via `provider/streaming.SSEReader`
- `Register()` for explicit provider registration
- `NewAzure` factory constructor
