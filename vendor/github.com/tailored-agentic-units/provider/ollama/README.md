# provider/ollama

Ollama provider for the [TAU](https://github.com/tailored-agentic-units) ecosystem.

```
go get github.com/tailored-agentic-units/provider/ollama
```

## Supported Protocols

| Protocol | Endpoint |
|----------|----------|
| Chat | `/v1/chat/completions` |
| Vision | `/v1/chat/completions` |
| Tools | `/v1/chat/completions` |
| Embeddings | `/v1/embeddings` |

## Usage

```go
import (
    "github.com/tailored-agentic-units/provider"
    "github.com/tailored-agentic-units/provider/ollama"
    "github.com/tailored-agentic-units/protocol/config"
)

// Register with the global provider registry
ollama.Register()

// Create via registry
p, err := provider.Create(&config.ProviderConfig{
    Name: "ollama",
})
```

## Authentication

Optional — supports bearer token and API key authentication via provider options:

```json
{
    "auth_type": "bearer",
    "token": "your-token"
}
```

## Dependencies

- `github.com/tailored-agentic-units/provider` — Provider interface and BaseProvider
- `github.com/tailored-agentic-units/protocol` — Protocol constants and streaming types
