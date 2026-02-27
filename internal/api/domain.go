package api

import (
	"github.com/JaimeStill/herald/internal/classifications"
	"github.com/JaimeStill/herald/internal/documents"
	"github.com/JaimeStill/herald/internal/prompts"
)

// Domain holds all domain systems that comprise the API.
type Domain struct {
	Classifications classifications.System
	Documents       documents.System
	Prompts         prompts.System
}

// NewDomain creates all domain systems from the API runtime.
func NewDomain(runtime *Runtime) *Domain {
	docsSystem := documents.New(
		runtime.Database.Connection(),
		runtime.Storage,
		runtime.Logger,
		runtime.Pagination,
	)

	promptsSystem := prompts.New(
		runtime.Database.Connection(),
		runtime.Logger,
		runtime.Pagination,
	)

	classificationsSystem := classifications.New(
		runtime.Database.Connection(),
		runtime.Agent,
		runtime.Logger,
		runtime.Pagination,
		runtime.Storage,
		docsSystem,
		promptsSystem,
	)

	return &Domain{
		Classifications: classificationsSystem,
		Documents:       docsSystem,
		Prompts:         promptsSystem,
	}
}
