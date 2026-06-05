// Package request provides protocol request types for LLM API calls.
// Each request type encapsulates protocol-specific data, provider, format,
// and model configuration needed for execution.
//
// Request types implement the Request interface, providing consistent
// access to protocol, headers, marshaled body, provider, and model.
//
// Use clean constructors to create requests:
//
//	chatReq := request.NewChat(provider, fmt, model, messages, options)
//	visionReq := request.NewVision(provider, fmt, model, messages, images, visionOpts, options)
//	toolsReq := request.NewTools(provider, fmt, model, messages, tools, options)
//	embeddingsReq := request.NewEmbeddings(provider, fmt, model, input, options)
package request
