package openai

import (
	"encoding/json"
	"fmt"

	"github.com/tailored-agentic-units/protocol/response"
)

func (f *Format) parseChat(body []byte) (*response.Response, error) {
	var raw apiResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse chat response: %w", err)
	}

	resp := &response.Response{Role: "assistant"}

	if len(raw.Choices) > 0 {
		choice := raw.Choices[0]
		if choice.Message.Content != "" {
			resp.Content = append(resp.Content, response.TextBlock{
				Text: choice.Message.Content,
			})
		}
		resp.StopReason = choice.FinishReason
	}

	resp.Usage = f.mapUsage(raw.Usage)

	return resp, nil
}

func (f *Format) parseTools(body []byte) (*response.Response, error) {
	var raw apiResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse tools response: %w", err)
	}

	resp := &response.Response{Role: "assistant"}

	if len(raw.Choices) > 0 {
		choice := raw.Choices[0]

		if choice.Message.Content != "" {
			resp.Content = append(resp.Content, response.TextBlock{
				Text: choice.Message.Content,
			})
		}

		for _, tc := range choice.Message.ToolCalls {
			var input map[string]any
			if err := json.Unmarshal([]byte(tc.Function.Arguments), &input); err != nil {
				return nil, fmt.Errorf("failed to parse tool arguments: %w", err)
			}

			resp.Content = append(resp.Content, response.ToolUseBlock{
				ID:    tc.ID,
				Name:  tc.Function.Name,
				Input: input,
			})
		}

		resp.StopReason = choice.FinishReason
	}

	resp.Usage = f.mapUsage(raw.Usage)

	return resp, nil
}

func (f *Format) parseEmbeddings(body []byte) (*response.EmbeddingsResponse, error) {
	var raw embeddingsResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse embeddings response: %w", err)
	}

	embeddings := make([][]float64, len(raw.Data))
	for i, d := range raw.Data {
		embeddings[i] = d.Embedding
	}

	resp := &response.EmbeddingsResponse{
		Embeddings: embeddings,
		Model:      raw.Model,
		Usage:      f.mapUsage(raw.Usage),
	}

	return resp, nil
}

func (f *Format) mapUsage(usage *apiUsage) *response.TokenUsage {
	if usage == nil {
		return nil
	}

	return &response.TokenUsage{
		InputTokens:  usage.PromptTokens,
		OutputTokens: usage.CompletionTokens,
		TotalTokens:  usage.TotalTokens,
	}
}
