# provider/azure

Azure OpenAI provider for the [TAU](https://github.com/tailored-agentic-units) ecosystem.

```
go get github.com/tailored-agentic-units/provider/azure
```

## Supported Protocols

| Protocol | Endpoint |
|----------|----------|
| Chat | `/deployments/{deployment}/chat/completions` |
| Vision | `/deployments/{deployment}/chat/completions` |
| Tools | `/deployments/{deployment}/chat/completions` |
| Embeddings | `/deployments/{deployment}/embeddings` |

## Authentication

| Type | Description |
|------|-------------|
| `api_key` | Azure API key via `api-key` header |
| `bearer` | Bearer token via `Authorization` header |
| `managed_identity` | Azure Managed Identity with automatic token refresh |

## Usage

```go
import (
    "github.com/tailored-agentic-units/provider"
    "github.com/tailored-agentic-units/provider/azure"
    "github.com/tailored-agentic-units/protocol/config"
)

azure.Register()

p, err := provider.Create(&config.ProviderConfig{
    Name:    "azure",
    BaseURL: "https://my-resource.openai.azure.com/openai",
    Options: map[string]any{
        "deployment":  "gpt-4o",
        "auth_type":   "api_key",
        "token":       "your-api-key",
        "api_version": "2024-06-01",
    },
})
```

## Dependencies

- `github.com/tailored-agentic-units/provider` — Provider interface and BaseProvider
- `github.com/tailored-agentic-units/protocol` — Protocol constants and streaming types
- `github.com/Azure/azure-sdk-for-go/sdk/azcore` — Azure core types
- `github.com/Azure/azure-sdk-for-go/sdk/azidentity` — Managed identity credentials
