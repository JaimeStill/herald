package openai

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"maps"

	"github.com/tailored-agentic-units/format"
	"github.com/tailored-agentic-units/protocol"
)

func (f *Format) marshalChat(data any) ([]byte, error) {
	d, ok := data.(*format.ChatData)
	if !ok {
		return nil, fmt.Errorf("expected *ChatData, got %T", data)
	}

	combined := make(map[string]any)
	combined["model"] = d.Model
	combined["messages"] = d.Messages
	maps.Copy(combined, d.Options)
	return json.Marshal(combined)
}

func (f *Format) marshalVision(data any) ([]byte, error) {
	d, ok := data.(*format.VisionData)
	if !ok {
		return nil, fmt.Errorf("expected *VisionData, got %T", data)
	}

	if len(d.Messages) == 0 {
		return nil, fmt.Errorf("messages cannot be empty for vision requests")
	}

	if len(d.Images) == 0 {
		return nil, fmt.Errorf("images cannot be empty for vision requests")
	}

	lastIdx := len(d.Messages) - 1
	message := d.Messages[lastIdx]

	var textContent string
	switch v := message.Content.(type) {
	case string:
		textContent = v
	default:
		return nil, fmt.Errorf("message content must be a string for vision transformation")
	}

	content := []map[string]any{
		{"type": "text", "text": textContent},
	}

	for _, img := range d.Images {
		var url string
		if img.URL != "" {
			url = img.URL
		} else {
			url = fmt.Sprintf(
				"data:image/%s;base64,%s",
				img.Format,
				base64.StdEncoding.EncodeToString(img.Data),
			)
		}

		imageURL := map[string]any{
			"url": url,
		}

		if d.VisionOptions != nil {
			maps.Copy(imageURL, d.VisionOptions)
		}

		content = append(content, map[string]any{
			"type":      "image_url",
			"image_url": imageURL,
		})
	}

	transformedMessages := make([]protocol.Message, len(d.Messages))
	copy(transformedMessages, d.Messages)
	transformedMessages[lastIdx] = protocol.Message{
		Role:    message.Role,
		Content: content,
	}

	combined := make(map[string]any)
	combined["model"] = d.Model
	combined["messages"] = transformedMessages
	maps.Copy(combined, d.Options)

	return json.Marshal(combined)
}

func (f *Format) marshalTools(data any) ([]byte, error) {
	d, ok := data.(*format.ToolsData)
	if !ok {
		return nil, fmt.Errorf("expected *ToolsData, got %T", data)
	}

	combined := make(map[string]any)
	combined["model"] = d.Model
	combined["messages"] = d.Messages

	tools := make([]map[string]any, len(d.Tools))
	for i, tool := range d.Tools {
		tools[i] = map[string]any{
			"type": "function",
			"function": map[string]any{
				"name":        tool.Name,
				"description": tool.Description,
				"parameters":  tool.Parameters,
			},
		}
	}

	combined["tools"] = tools

	maps.Copy(combined, d.Options)
	return json.Marshal(combined)
}

func (f *Format) marshalEmbeddings(data any) ([]byte, error) {
	d, ok := data.(*format.EmbeddingsData)
	if !ok {
		return nil, fmt.Errorf("expected *EmbeddingsData, got %T", data)
	}

	combined := make(map[string]any)
	combined["model"] = d.Model
	combined["input"] = d.Input
	maps.Copy(combined, d.Options)
	return json.Marshal(combined)
}
