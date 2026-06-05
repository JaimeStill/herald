# tau/agent

High-level interface for LLM interactions with protocol-specific methods for chat, vision, tools, and embeddings.

## Install

```bash
go get github.com/tailored-agentic-units/agent
```

## Overview

The agent package wraps the client layer with convenient methods for common LLM operations. Agents are created via explicit dependency injection, receiving a provider and format instance from the caller.

```go
p, _ := provider.Create(cfg.Provider)
f, _ := format.Create(cfg.Format)
a := agent.New(cfg, p, f)

resp, err := a.Chat(ctx, messages)
```

## Packages

| Package | Description |
|---------|-------------|
| `agent` | Agent interface and constructor |
| `client` | HTTP client with retry logic and health tracking |
| `request` | Protocol request types (chat, vision, tools, embeddings) |
| `registry` | Named agent configuration with lazy instantiation |
| `mock` | Test doubles for agent, client, provider, and format |

## License

See repository root.
