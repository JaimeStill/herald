package agent

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/tailored-agentic-units/protocol/config"
)

// ErrorType categorizes agent errors by their source.
type ErrorType string

const (
	// ErrorTypeInit indicates initialization or configuration errors.
	ErrorTypeInit ErrorType = "init"

	// ErrorTypeLLM indicates errors from LLM interactions.
	ErrorTypeLLM ErrorType = "llm"
)

// Sentinel errors for the agent registry.
var (
	ErrAgentNotFound  = errors.New("agent not found")
	ErrAgentExists    = errors.New("agent already registered")
	ErrEmptyAgentName = errors.New("agent name is empty")
)

// AgentError provides detailed error information for agent operations.
// Includes error categorization, unique identification, and contextual metadata.
type AgentError struct {
	// Type categorizes the error (init or llm).
	Type ErrorType `json:"type"`

	// ID is a unique identifier for this error instance.
	ID uuid.UUID `json:"uuid,omitempty"`

	// Name identifies the agent that generated the error.
	Name string `json:"name,omitempty"`

	// Code is an application-specific error code.
	Code string `json:"code,omitempty"`

	// Message describes what went wrong.
	Message string `json:"message"`

	// Cause is the underlying error that caused this error.
	Cause error `json:"-"`

	// Client identifies the provider/model combination.
	Client string `json:"client,omitempty"`

	// Timestamp records when the error occurred.
	Timestamp time.Time `json:"timestamp"`
}

// NewAgentError creates a new AgentError with the specified type and message.
// Optional ErrorOption functions can be provided to set additional fields.
func NewAgentError(errorType ErrorType, message string, options ...ErrorOption) *AgentError {
	err := &AgentError{
		Type:      errorType,
		Message:   message,
		Timestamp: time.Now(),
	}

	for _, option := range options {
		option(err)
	}

	return err
}

// Error returns a formatted error message.
// Format varies based on available context (client, name).
func (e *AgentError) Error() string {
	if e.Client != "" && e.Name != "" {
		return fmt.Sprintf("Agent error [%s/%s]: %s", e.Client, e.Name, e.Message)
	}
	if e.Name != "" {
		return fmt.Sprintf("Agent error [%s]: %s", e.Name, e.Message)
	}

	return fmt.Sprintf("Agent error: %s", e.Message)
}

// Unwrap returns the underlying cause error.
// Implements the error unwrapping interface for errors.Is and errors.As.
func (e *AgentError) Unwrap() error {
	return e.Cause
}

// ErrorOption is a function that modifies an AgentError.
// Used with NewAgentError to set optional fields.
type ErrorOption func(*AgentError)

// WithCode sets the error code.
func WithCode(code string) ErrorOption {
	return func(e *AgentError) {
		e.Code = code
	}
}

// WithCause sets the underlying cause error.
func WithCause(cause error) ErrorOption {
	return func(e *AgentError) {
		e.Cause = cause
	}
}

// WithName sets the agent name that generated the error.
func WithName(name string) ErrorOption {
	return func(e *AgentError) {
		e.Name = name
	}
}

// WithAgent extracts identification from agent configuration.
// Creates a string in the format "provider/model", "provider", or "model"
// depending on available information.
func WithAgent(cfg *config.AgentConfig) ErrorOption {
	return func(e *AgentError) {
		providerName := ""
		modelName := ""

		if cfg.Provider != nil {
			providerName = cfg.Provider.Name
		}
		if cfg.Model != nil {
			modelName = cfg.Model.Name
		}

		if providerName != "" && modelName != "" {
			e.Client = fmt.Sprintf("%s/%s", providerName, modelName)
		} else if providerName != "" {
			e.Client = providerName
		} else if modelName != "" {
			e.Client = modelName
		} else {
			e.Client = "unknown"
		}
	}
}

// WithID sets a unique identifier for this error instance.
func WithID(id uuid.UUID) ErrorOption {
	return func(e *AgentError) {
		e.ID = id
	}
}

// NewAgentInitError creates an initialization error.
// Shorthand for NewAgentError(ErrorTypeInit, message, options...).
func NewAgentInitError(message string, options ...ErrorOption) *AgentError {
	return NewAgentError(ErrorTypeInit, message, options...)
}

// NewAgentLLMError creates an LLM interaction error.
// Shorthand for NewAgentError(ErrorTypeLLM, message, options...).
func NewAgentLLMError(message string, options ...ErrorOption) *AgentError {
	return NewAgentError(ErrorTypeLLM, message, options...)
}
