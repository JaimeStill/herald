package openai

import (
	"encoding/json"
	"fmt"

	"github.com/tailored-agentic-units/format"
	"github.com/tailored-agentic-units/protocol"
	"github.com/tailored-agentic-units/protocol/response"
)

// Register registers the OpenAI format with the global format registry.
func Register() {
	format.Register("openai", Factory)
}

// Factory creates a new Format instance for use with the format registry.
func Factory() (format.Format, error) {
	return &Format{}, nil
}

// Format implements format.Format for OpenAI-compatible APIs.
type Format struct{}

func (f *Format) Name() string {
	return "openai"
}

func (f *Format) Marshal(proto protocol.Protocol, data any) ([]byte, error) {
	switch proto {
	case protocol.Chat:
		return f.marshalChat(data)
	case protocol.Vision:
		return f.marshalVision(data)
	case protocol.Tools:
		return f.marshalTools(data)
	case protocol.Embeddings:
		return f.marshalEmbeddings(data)
	default:
		return nil, fmt.Errorf("unsupported protocol: %s", proto)
	}
}

func (f *Format) Parse(proto protocol.Protocol, body []byte) (any, error) {
	switch proto {
	case protocol.Chat, protocol.Vision:
		return f.parseChat(body)
	case protocol.Tools:
		return f.parseTools(body)
	case protocol.Embeddings:
		return f.parseEmbeddings(body)
	default:
		return nil, fmt.Errorf("unsupported protocol: %s", proto)
	}
}

func (f *Format) ParseStreamChunk(proto protocol.Protocol, data []byte) (*response.StreamingResponse, error) {
	if proto == protocol.Embeddings {
		return nil, fmt.Errorf("protocol %s does not support streaming", proto)
	}

	var raw streamChunk
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse streaming chunk: %w", err)
	}

	chunk := &response.StreamingResponse{}

	if len(raw.Choices) > 0 {
		choice := raw.Choices[0]
		if choice.Delta.Content != "" {
			chunk.Content = append(chunk.Content, response.TextBlock{
				Text: choice.Delta.Content,
			})
		}
		if choice.FinishReason != nil {
			chunk.StopReason = *choice.FinishReason
		}
	}

	return chunk, nil
}
