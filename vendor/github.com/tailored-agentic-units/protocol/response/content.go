package response

// ContentBlock represents a single block of content within a model response.
type ContentBlock interface {
	blockType() string
}

// TextBlock is a ContentBlock containing plain text output from the model.
type TextBlock struct {
	Text string
}

func (b TextBlock) blockType() string { return "text" }

// ToolUseBlock is a ContentBlock representing a tool invocation requested by the model.
type ToolUseBlock struct {
	ID    string
	Name  string
	Input map[string]any
}

func (b ToolUseBlock) blockType() string { return "tool_use" }
