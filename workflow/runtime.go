package workflow

import (
	"log/slog"

	"github.com/JaimeStill/herald/internal/documents"
	"github.com/JaimeStill/herald/internal/prompts"
	"github.com/JaimeStill/herald/pkg/storage"

	gaconfig "github.com/JaimeStill/go-agents/pkg/config"
)

// Runtime bundles the dependencies that workflow nodes require.
// It is constructed by higher-level composition code from Infrastructure and Domain systems.
type Runtime struct {
	Agent     gaconfig.AgentConfig
	Storage   storage.System
	Documents documents.System
	Prompts   prompts.System
	Logger    *slog.Logger
}
