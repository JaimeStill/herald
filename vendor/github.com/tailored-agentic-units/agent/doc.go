// Package agent provides a high-level interface for LLM interactions.
// It wraps the client layer with convenient methods for common operations
// like chat, vision, tools, and embeddings.
//
// # Agent Interface
//
// The Agent interface provides protocol-specific methods that accept
// pre-built message slices, allowing callers to manage system prompts
// and conversation history externally:
//
//	type Agent interface {
//	    ID() string
//	    Client() client.Client
//	    Provider() provider.Provider
//	    Format() format.Format
//	    Model() *model.Model
//
//	    Chat(ctx, messages, opts...) (*response.Response, error)
//	    ChatStream(ctx, messages, opts...) (<-chan *response.StreamingResponse, error)
//	    Vision(ctx, messages, images, opts...) (*response.Response, error)
//	    VisionStream(ctx, messages, images, opts...) (<-chan *response.StreamingResponse, error)
//	    Tools(ctx, messages, tools, opts...) (*response.Response, error)
//	    Embed(ctx, input, opts...) (*response.EmbeddingsResponse, error)
//	}
//
// # Creating an Agent
//
// Agents are created via explicit dependency injection — the caller provides
// the provider and format instances:
//
//	p, _ := provider.Create(cfg.Provider)
//	f, _ := format.Create(cfg.Format)
//	a := agent.New(cfg, p, f)
//
// # Options Management
//
// All protocol methods accept optional parameters that are merged with
// model defaults. Request options take precedence over model defaults.
//
// # Error Handling
//
// The package provides AgentError with categorization (init, llm),
// unique identification, and contextual metadata for structured error reporting.
//
// # Thread Safety
//
// Agents are safe for concurrent use. Multiple goroutines can call protocol
// methods simultaneously on the same agent instance.
package agent
