package request

import (
	"github.com/tailored-agentic-units/format"
	"github.com/tailored-agentic-units/protocol"
	"github.com/tailored-agentic-units/protocol/model"
	"github.com/tailored-agentic-units/provider"
)

// VisionRequest represents a vision protocol request with image inputs.
// Separates images and vision-specific options from model configuration options.
type VisionRequest struct {
	messages      []protocol.Message
	images        []format.Image
	visionOptions map[string]any
	options       map[string]any
	prov          provider.Provider
	fmt           format.Format
	mdl           *model.Model
}

// NewVision creates a new VisionRequest with the given components.
// Messages contain the conversation history.
// Images are image attachments to analyze.
// VisionOptions are vision-specific settings (e.g., detail level).
// Options specify model configuration (temperature, max_tokens, etc.).
func NewVision(p provider.Provider, f format.Format, m *model.Model, messages []protocol.Message, images []format.Image, visionOpts, opts map[string]any) *VisionRequest {
	return &VisionRequest{
		messages:      messages,
		images:        images,
		visionOptions: visionOpts,
		options:       opts,
		prov:          p,
		fmt:           f,
		mdl:           m,
	}
}

// Protocol returns the Vision protocol identifier.
func (r *VisionRequest) Protocol() protocol.Protocol {
	return protocol.Vision
}

// Headers returns the HTTP headers for a vision request.
func (r *VisionRequest) Headers() map[string]string {
	return map[string]string{
		"Content-Type": "application/json",
	}
}

// Marshal delegates to the format for wire-format-specific JSON encoding.
func (r *VisionRequest) Marshal() ([]byte, error) {
	return r.fmt.Marshal(protocol.Vision, &format.VisionData{
		Model:         r.mdl.Name,
		Messages:      r.messages,
		Images:        r.images,
		VisionOptions: r.visionOptions,
		Options:       r.options,
	})
}

// Provider returns the provider for this request.
func (r *VisionRequest) Provider() provider.Provider {
	return r.prov
}

// Model returns the model for this request.
func (r *VisionRequest) Model() *model.Model {
	return r.mdl
}

// Format returns the wire format for response parsing.
func (r *VisionRequest) Format() format.Format {
	return r.fmt
}
