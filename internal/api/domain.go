package api

import "github.com/JaimeStill/herald/internal/documents"

// Domain holds all domain systems that comprise the API.
type Domain struct {
	Documents documents.System
}

// NewDomain creates all domain systems from the API runtime.
func NewDomain(runtime *Runtime) *Domain {
	docsSystem := documents.New(
		runtime.Database.Connection(),
		runtime.Storage,
		runtime.Logger,
		runtime.Pagination,
	)

	return &Domain{
		Documents: docsSystem,
	}
}
