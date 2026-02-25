package prompts

import (
	"context"

	"github.com/google/uuid"

	"github.com/JaimeStill/herald/pkg/pagination"
)

// System defines the public contract for prompt domain operations.
type System interface {
	Handler() *Handler

	List(
		ctx context.Context,
		page pagination.PageRequest,
		filters Filters,
	) (*pagination.PageResult[Prompt], error)

	Find(ctx context.Context, id uuid.UUID) (*Prompt, error)
	Create(ctx context.Context, cmd CreateCommand) (*Prompt, error)
	Update(ctx context.Context, id uuid.UUID, cmd UpdateCommand) (*Prompt, error)
	Delete(ctx context.Context, id uuid.UUID) error
	Activate(ctx context.Context, id uuid.UUID) (*Prompt, error)
	Deactivate(ctx context.Context, id uuid.UUID) (*Prompt, error)
}
