# format/openai

OpenAI-compatible wire format for the [TAU](https://github.com/tailored-agentic-units) ecosystem.

```
go get github.com/tailored-agentic-units/format/openai
```

## Supported Protocols

| Protocol | Marshal | Parse | Stream |
|----------|---------|-------|--------|
| Chat | yes | yes | yes |
| Vision | yes | yes | yes |
| Tools | yes | yes | yes |
| Embeddings | yes | yes | no |

## Usage

```go
import (
    "github.com/tailored-agentic-units/format/openai"
)

// Register with the global format registry
openai.Register()

// Or use directly
f, _ := openai.Factory()
body, err := f.Marshal(protocol.Chat, &format.ChatData{
    Model:    "gpt-4o",
    Messages: messages,
    Options:  map[string]any{"temperature": 0.7},
})
```

## Dependencies

- `github.com/tailored-agentic-units/format` — Format interface and data types
- `github.com/tailored-agentic-units/protocol` — Protocol constants and message types
