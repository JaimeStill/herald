package response

import "strings"

// Response represents a complete model response containing content blocks, stop reason, and token usage.
type Response struct {
	Role       string
	Content    []ContentBlock
	StopReason string
	Usage      *TokenUsage
}

// Text returns the concatenated text from all TextBlock content blocks.
func (r *Response) Text() string {
	var text strings.Builder
	for _, block := range r.Content {
		if tb, ok := block.(TextBlock); ok {
			text.WriteString(tb.Text)
		}
	}
	return text.String()
}

// ToolCalls returns all ToolUseBlock content blocks from the response.
func (r *Response) ToolCalls() []ToolUseBlock {
	var calls []ToolUseBlock
	for _, block := range r.Content {
		if tb, ok := block.(ToolUseBlock); ok {
			calls = append(calls, tb)
		}
	}
	return calls
}
