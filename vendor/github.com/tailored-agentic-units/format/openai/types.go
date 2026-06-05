package openai

// Wire types for OpenAI API request/response serialization.

type apiChoice struct {
	Message      apiMessage `json:"message"`
	FinishReason string     `json:"finish_reason,omitempty"`
}

type apiDelta struct {
	Content string `json:"content"`
}

type apiEmbedding struct {
	Embedding []float64 `json:"embedding"`
}

type embeddingsResponse struct {
	Data  []apiEmbedding `json:"data"`
	Model string         `json:"model"`
	Usage *apiUsage      `json:"usage,omitempty"`
}

type apiMessage struct {
	Role      string        `json:"role"`
	Content   string        `json:"content"`
	ToolCalls []apiToolCall `json:"tool_calls,omitempty"`
}

type apiResponse struct {
	Choices []apiChoice `json:"choices"`
	Usage   *apiUsage   `json:"usage,omitempty"`
}

type streamChoice struct {
	Delta        apiDelta `json:"delta"`
	FinishReason *string  `json:"finish_reason"`
}

type streamChunk struct {
	Choices []streamChoice `json:"choices"`
}

type apiToolCall struct {
	ID       string          `json:"id"`
	Type     string          `json:"type"`
	Function apiToolFunction `json:"function"`
}

type apiToolFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type apiUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}
