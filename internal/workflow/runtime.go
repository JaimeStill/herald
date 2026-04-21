package workflow

import (
	"context"
	"log/slog"

	"github.com/tailored-agentic-units/agent"

	"github.com/JaimeStill/herald/internal/documents"
	"github.com/JaimeStill/herald/internal/prompts"
	"github.com/JaimeStill/herald/pkg/storage"
)

// Runtime bundles the dependencies that workflow nodes require.
// It is constructed by higher-level composition code from Infrastructure and Domain systems.
type Runtime struct {
	NewAgent  func(ctx context.Context) (agent.Agent, error)
	Model     string
	Provider  string
	Storage   storage.System
	Documents documents.System
	Prompts   prompts.System
	Logger    *slog.Logger
}
