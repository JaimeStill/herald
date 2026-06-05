package agent

import (
	"context"
	"fmt"
	"maps"

	"github.com/google/uuid"
	"github.com/tailored-agentic-units/agent/client"
	"github.com/tailored-agentic-units/agent/request"
	"github.com/tailored-agentic-units/format"
	"github.com/tailored-agentic-units/protocol"
	"github.com/tailored-agentic-units/protocol/config"
	"github.com/tailored-agentic-units/protocol/model"
	"github.com/tailored-agentic-units/protocol/response"
	"github.com/tailored-agentic-units/provider"
)

// Agent provides a high-level interface for LLM interactions.
// Methods are protocol-specific and handle message formatting
// and response type assertions.
//
// Each agent has a unique identifier that remains stable across its lifetime.
// The ID is used for orchestration scenarios including hub registration,
// message routing, lifecycle tracking, and distributed tracing.
type Agent interface {
	// ID returns the unique identifier for the agent.
	// The ID is assigned at creation time using UUIDv7 and never changes.
	ID() string

	// Client returns the underlying HTTP client.
	Client() client.Client

	// Provider returns the provider instance.
	Provider() provider.Provider

	// Format returns the wire format instance.
	Format() format.Format

	// Model returns the model instance.
	Model() *model.Model

	// Chat executes a chat protocol request.
	// Returns the parsed response or an error.
	Chat(ctx context.Context, messages []protocol.Message, opts ...map[string]any) (*response.Response, error)

	// ChatStream executes a streaming chat protocol request.
	// Automatically sets stream: true in options.
	// Returns a channel of streaming responses or an error.
	ChatStream(ctx context.Context, messages []protocol.Message, opts ...map[string]any) (<-chan *response.StreamingResponse, error)

	// Vision executes a vision protocol request with images.
	// Returns the parsed response or an error.
	Vision(ctx context.Context, messages []protocol.Message, images []format.Image, opts ...map[string]any) (*response.Response, error)

	// VisionStream executes a streaming vision protocol request with images.
	// Returns a channel of streaming responses or an error.
	VisionStream(ctx context.Context, messages []protocol.Message, images []format.Image, opts ...map[string]any) (<-chan *response.StreamingResponse, error)

	// Tools executes a tools protocol request with function definitions.
	// Returns the parsed response with tool calls or an error.
	Tools(ctx context.Context, messages []protocol.Message, tools []format.ToolDefinition, opts ...map[string]any) (*response.Response, error)

	// Embed executes an embeddings protocol request.
	// Returns the parsed embeddings response or an error.
	Embed(ctx context.Context, input string, opts ...map[string]any) (*response.EmbeddingsResponse, error)
}

// agent implements the Agent interface.
type agent struct {
	id       string
	client   client.Client
	provider provider.Provider
	fmt      format.Format
	model    *model.Model
}

// New creates a new Agent with explicit dependency injection.
// The provider and format are passed in directly rather than resolved from config.
// Assigns a unique UUIDv7 identifier for orchestration and tracking.
func New(cfg *config.AgentConfig, p provider.Provider, f format.Format) Agent {
	m := model.New(cfg.Model)
	c := client.New(cfg.Client)

	return &agent{
		id:       uuid.Must(uuid.NewV7()).String(),
		client:   c,
		provider: p,
		fmt:      f,
		model:    m,
	}
}

func (a *agent) ID() string {
	return a.id
}

func (a *agent) Client() client.Client {
	return a.client
}

func (a *agent) Provider() provider.Provider {
	return a.provider
}

func (a *agent) Format() format.Format {
	return a.fmt
}

func (a *agent) Model() *model.Model {
	return a.model
}

// Chat executes a chat protocol request.
// Merges model's configured chat options with runtime opts.
// Returns parsed Response or error.
func (a *agent) Chat(ctx context.Context, messages []protocol.Message, opts ...map[string]any) (*response.Response, error) {
	options := a.mergeOptions(protocol.Chat, opts...)

	req := request.NewChat(a.provider, a.fmt, a.model, messages, options)

	result, err := a.client.Execute(ctx, req)
	if err != nil {
		return nil, err
	}

	resp, ok := result.(*response.Response)
	if !ok {
		return nil, fmt.Errorf("unexpected response type: %T", result)
	}

	return resp, nil
}

// ChatStream executes a streaming chat protocol request.
// Merges model's configured chat options with runtime opts.
// Automatically sets stream: true in options.
// Returns a channel of StreamingResponse or error.
func (a *agent) ChatStream(ctx context.Context, messages []protocol.Message, opts ...map[string]any) (<-chan *response.StreamingResponse, error) {
	options := a.mergeOptions(protocol.Chat, opts...)
	options["stream"] = true

	req := request.NewChat(a.provider, a.fmt, a.model, messages, options)

	return a.client.ExecuteStream(ctx, req)
}

// Vision executes a vision protocol request with images.
// Merges model's configured vision options with runtime opts.
// Extracts vision_options from opts if present.
// Returns parsed Response or error.
func (a *agent) Vision(ctx context.Context, messages []protocol.Message, images []format.Image, opts ...map[string]any) (*response.Response, error) {
	options := a.mergeOptions(protocol.Vision, opts...)

	var visionOptions map[string]any
	if vOpts, exists := options["vision_options"]; exists {
		if vOptsMap, ok := vOpts.(map[string]any); ok {
			visionOptions = vOptsMap
			delete(options, "vision_options")
		}
	}

	req := request.NewVision(a.provider, a.fmt, a.model, messages, images, visionOptions, options)

	result, err := a.client.Execute(ctx, req)
	if err != nil {
		return nil, err
	}

	resp, ok := result.(*response.Response)
	if !ok {
		return nil, fmt.Errorf("unexpected response type: %T", result)
	}

	return resp, nil
}

// VisionStream executes a streaming vision protocol request with images.
// Merges model's configured vision options with runtime opts.
// Automatically sets stream: true in options.
// Returns a channel of StreamingResponse or error.
func (a *agent) VisionStream(ctx context.Context, messages []protocol.Message, images []format.Image, opts ...map[string]any) (<-chan *response.StreamingResponse, error) {
	options := a.mergeOptions(protocol.Vision, opts...)
	options["stream"] = true

	var visionOptions map[string]any
	if vOpts, exists := options["vision_options"]; exists {
		if vOptsMap, ok := vOpts.(map[string]any); ok {
			visionOptions = vOptsMap
			delete(options, "vision_options")
		}
	}

	req := request.NewVision(a.provider, a.fmt, a.model, messages, images, visionOptions, options)

	return a.client.ExecuteStream(ctx, req)
}

// Tools executes a tools protocol request with function definitions.
// Merges model's configured tools options with runtime opts.
// Returns parsed Response with tool calls or error.
func (a *agent) Tools(ctx context.Context, messages []protocol.Message, tools []format.ToolDefinition, opts ...map[string]any) (*response.Response, error) {
	options := a.mergeOptions(protocol.Tools, opts...)

	req := request.NewTools(a.provider, a.fmt, a.model, messages, tools, options)

	result, err := a.client.Execute(ctx, req)
	if err != nil {
		return nil, err
	}

	resp, ok := result.(*response.Response)
	if !ok {
		return nil, fmt.Errorf("unexpected response type: %T", result)
	}

	return resp, nil
}

// Embed executes an embeddings protocol request.
// Merges model's configured embeddings options with runtime opts.
// Returns parsed EmbeddingsResponse or error.
func (a *agent) Embed(ctx context.Context, input string, opts ...map[string]any) (*response.EmbeddingsResponse, error) {
	options := a.mergeOptions(protocol.Embeddings, opts...)

	req := request.NewEmbeddings(a.provider, a.fmt, a.model, input, options)

	result, err := a.client.Execute(ctx, req)
	if err != nil {
		return nil, err
	}

	resp, ok := result.(*response.EmbeddingsResponse)
	if !ok {
		return nil, fmt.Errorf("unexpected response type: %T", result)
	}

	return resp, nil
}

// mergeOptions creates options by merging model defaults with runtime options.
func (a *agent) mergeOptions(proto protocol.Protocol, opts ...map[string]any) map[string]any {
	options := make(map[string]any)
	if modelOpts := a.model.Options[proto]; modelOpts != nil {
		maps.Copy(options, modelOpts)
	}
	if len(opts) > 0 && opts[0] != nil {
		maps.Copy(options, opts[0])
	}
	return options
}
