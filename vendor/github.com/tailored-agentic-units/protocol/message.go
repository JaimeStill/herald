package protocol

// Role identifies the sender of a conversation message.
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

// ToolFunction holds the name and arguments for a function call.
type ToolFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// ToolCall represents a tool invocation requested by the model.
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

// NewToolCall creates a ToolCall with type "function".
func NewToolCall(id, name, arguments string) ToolCall {
	return ToolCall{
		ID:   id,
		Type: "function",
		Function: ToolFunction{
			Name:      name,
			Arguments: arguments,
		},
	}
}

// Message represents a single message in a conversation.
// Role indicates the sender, and Content can be a string for text or a
// structured object for multimodal content (e.g., vision arrays).
//
// For tool-calling conversations, assistant messages carry ToolCalls and
// tool result messages carry a ToolCallID that correlates back to the request.
type Message struct {
	Role       Role       `json:"role"`
	Content    any        `json:"content"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
}

// NewMessage creates a Message with the given role and content.
func NewMessage(role Role, content any) Message {
	return Message{Role: role, Content: content}
}

// UserMessage creates a user message with text content.
func UserMessage(content string) Message {
	return Message{Role: RoleUser, Content: content}
}

// SystemMessage creates a system message with text content.
func SystemMessage(content string) Message {
	return Message{Role: RoleSystem, Content: content}
}

// AssistantMessage creates an assistant message with text content.
func AssistantMessage(content string) Message {
	return Message{Role: RoleAssistant, Content: content}
}

// ToolMessage creates a tool result message correlating to a specific tool call.
func ToolMessage(toolCallID string, content string) Message {
	return Message{Role: RoleTool, Content: content, ToolCallID: toolCallID}
}

// InitMessages creates a single-element message slice from a role and content string.
// Convenience wrapper for the common pattern of initializing a conversation from a prompt.
func InitMessages(role Role, content string) []Message {
	return []Message{NewMessage(role, content)}
}
