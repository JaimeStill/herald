package classifications

import (
	"context"

	"github.com/google/uuid"

	"github.com/JaimeStill/herald/pkg/pagination"
)

// System defines the public contract for classification domain operations.
type System interface {
	Handler() *Handler

	List(
		ctx context.Context,
		page pagination.PageRequest,
		filters Filters,
	) (*pagination.PageResult[Classification], error)

	Find(ctx context.Context, id uuid.UUID) (*Classification, error)
	FindByDocument(ctx context.Context, documentID uuid.UUID) (*Classification, error)
	Classify(ctx context.Context, documentID uuid.UUID) (*Classification, error)
	Validate(ctx context.Context, id uuid.UUID, cmd ValidateCommand) (*Classification, error)
	Update(ctx context.Context, id uuid.UUID, cmd UpdateCommand) (*Classification, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
