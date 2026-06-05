// Package response provides unified response types for LLM interactions.
//
// The Response type uses a ContentBlock interface to represent different types
// of model output (text, tool use) in a single response structure, replacing
// the legacy separate ChatResponse/ToolsResponse pattern.
package response
