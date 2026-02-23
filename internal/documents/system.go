package documents

import (
	"context"

	"github.com/google/uuid"

	"github.com/JaimeStill/herald/pkg/pagination"
)

// System defines the public contract for document domain operations.
type System interface {
	Handler(maxUploadSize int64) *Handler

	List(
		ctx context.Context,
		page pagination.PageRequest,
		filters Filters,
	) (*pagination.PageResult[Document], error)

	Find(ctx context.Context, id uuid.UUID) (*Document, error)
	Create(ctx context.Context, cmd CreateCommand) (*Document, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
