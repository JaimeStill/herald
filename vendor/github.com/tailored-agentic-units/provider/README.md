# provider

LLM provider abstraction for the [TAU](https://github.com/tailored-agentic-units) ecosystem.

```
go get github.com/tailored-agentic-units/provider
```

## Architecture

The root module exposes the `Provider` interface, `BaseProvider`, `Request` type, and a thread-safe registry. Concrete implementations live in sub-modules with their own `go.mod`, keeping cloud SDK dependencies isolated.

```
provider (root)       — Provider interface, registry, BaseProvider, Request
  streaming/          — SSEReader (stdlib only)
  ollama/             — Ollama provider (sub-module)
  azure/              — Azure OpenAI provider (sub-module)
  bedrock/            — AWS Bedrock provider (sub-module)
```

## Sub-modules

| Module | Install | Auth |
|--------|---------|------|
| [ollama](./ollama/) | `go get github.com/tailored-agentic-units/provider/ollama` | Optional bearer/API key |
| [azure](./azure/) | `go get github.com/tailored-agentic-units/provider/azure` | API key, bearer, managed identity |
| [bedrock](./bedrock/) | `go get github.com/tailored-agentic-units/provider/bedrock` | Default chain, static, profile (SigV4) |

## Registration

Sub-modules use explicit registration — no hidden `init()` side effects:

```go
import (
    "github.com/tailored-agentic-units/provider"
    "github.com/tailored-agentic-units/provider/ollama"
    "github.com/tailored-agentic-units/protocol/config"
)

func main() {
    ollama.Register()
    p, err := provider.Create(&config.ProviderConfig{
        Name: "ollama",
    })
}
```

## Dependencies

- Root: `github.com/tailored-agentic-units/protocol`
- Sub-modules: root + protocol + cloud SDKs (Azure SDK, AWS SDK)
