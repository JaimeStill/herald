package prompts

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/JaimeStill/herald/pkg/pagination"
	"github.com/JaimeStill/herald/pkg/query"
	"github.com/JaimeStill/herald/pkg/repository"
)

type repo struct {
	db         *sql.DB
	logger     *slog.Logger
	pagination pagination.Config
}

// New creates a prompt repository implementing the System interface.
func New(
	db *sql.DB,
	logger *slog.Logger,
	pagination pagination.Config,
) System {
	return &repo{
		db:         db,
		logger:     logger.With("system", "prompts"),
		pagination: pagination,
	}
}

func (r *repo) Handler() *Handler {
	return NewHandler(r, r.logger, r.pagination)
}

func (r *repo) List(
	ctx context.Context,
	page pagination.PageRequest,
	filters Filters,
) (*pagination.PageResult[Prompt], error) {
	page.Normalize(r.pagination)

	qb := query.
		NewBuilder(projection, defaultSort).
		WhereSearch(page.Search, "Name", "Description")

	filters.Apply(qb)

	if len(page.Sort) > 0 {
		qb.OrderByFields(page.Sort)
	}

	countSQL, countArgs := qb.BuildCount()
	var total int
	if err := r.db.QueryRowContext(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count prompts: %w", err)
	}

	pageSQL, pageArgs := qb.BuildPage(page.Page, page.PageSize)
	prompts, err := repository.QueryMany(ctx, r.db, pageSQL, pageArgs, scanPrompt)
	if err != nil {
		return nil, fmt.Errorf("query prompts: %w", err)
	}

	result := pagination.NewPageResult(prompts, total, page.Page, page.PageSize)
	return &result, nil
}

func (r *repo) Find(ctx context.Context, id uuid.UUID) (*Prompt, error) {
	q, args := query.NewBuilder(projection).BuildSingle("ID", id)

	p, err := repository.QueryOne(ctx, r.db, q, args, scanPrompt)
	if err != nil {
		return nil, repository.MapError(err, ErrNotFound, ErrDuplicate)
	}
	return &p, nil
}

func (r *repo) Create(ctx context.Context, cmd CreateCommand) (*Prompt, error) {
	q := `
		INSERT INTO prompts(name, stage, instructions, description)
		VALUES ($1, $2, $3, $4)
		RETURNING id, name, stage, instructions, description, active`

	args := []any{cmd.Name, cmd.Stage, cmd.Instructions, cmd.Description}

	p, err := repository.WithTx(ctx, r.db, func(tx *sql.Tx) (Prompt, error) {
		return repository.QueryOne(ctx, tx, q, args, scanPrompt)
	})

	if err != nil {
		return nil, repository.MapError(err, ErrNotFound, ErrDuplicate)
	}

	r.logger.Info("prompt created", "id", p.ID, "name", p.Name, "stage", p.Stage)
	return &p, nil
}

func (r *repo) Update(ctx context.Context, id uuid.UUID, cmd UpdateCommand) (*Prompt, error) {
	q := `
		UPDATE prompts
		SET name = $1, stage = $2, instructions = $3, description = $4
		WHERE id = $5
		RETURNING id, name, stage, instructions, description, active`

	args := []any{cmd.Name, cmd.Stage, cmd.Instructions, cmd.Description, id}

	p, err := repository.WithTx(ctx, r.db, func(tx *sql.Tx) (Prompt, error) {
		return repository.QueryOne(ctx, tx, q, args, scanPrompt)
	})

	if err != nil {
		return nil, repository.MapError(err, ErrNotFound, ErrDuplicate)
	}

	r.logger.Info("prompt updated", "id", p.ID, "name", p.Name)
	return &p, nil
}

func (r *repo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := repository.WithTx(ctx, r.db, func(tx *sql.Tx) (struct{}, error) {
		if err := repository.ExecExpectOne(
			ctx, tx,
			"DELETE FROM prompts WHERE id = $1",
			id,
		); err != nil {
			return struct{}{}, err
		}
		return struct{}{}, nil
	})

	if err != nil {
		return repository.MapError(err, ErrNotFound, ErrDuplicate)
	}

	r.logger.Info("prompt deleted", "id", id)
	return nil
}

func (r *repo) Activate(ctx context.Context, id uuid.UUID) (*Prompt, error) {
	p, err := repository.WithTx(ctx, r.db, func(tx *sql.Tx) (Prompt, error) {
		findQ, findArgs := query.NewBuilder(projection).BuildSingle("ID", id)
		target, err := repository.QueryOne(ctx, tx, findQ, findArgs, scanPrompt)
		if err != nil {
			return Prompt{}, err
		}

		_, err = tx.ExecContext(
			ctx,
			"UPDATE prompts SET active = false WHERE stage = $1 AND active = true",
			target.Stage,
		)
		if err != nil {
			return Prompt{}, fmt.Errorf("deactivate current: %w", err)
		}

		activateQ := `
			UPDATE prompts SET active = true
			WHERE id = $1
			RETURNING id, name, stage, instructions, description, active`

		return repository.QueryOne(ctx, tx, activateQ, []any{id}, scanPrompt)
	})

	if err != nil {
		return nil, repository.MapError(err, ErrNotFound, ErrDuplicate)
	}

	r.logger.Info("prompt activated", "id", p.ID, "name", p.Name, "stage", p.Stage)
	return &p, nil
}

func (r *repo) Deactivate(ctx context.Context, id uuid.UUID) (*Prompt, error) {
	q := `
		UPDATE prompts SET active = false
		WHERE id = $1
		RETURNING id, name, stage, instructions, description, active`

	p, err := repository.WithTx(ctx, r.db, func(tx *sql.Tx) (Prompt, error) {
		return repository.QueryOne(ctx, tx, q, []any{id}, scanPrompt)
	})

	if err != nil {
		return nil, repository.MapError(err, ErrNotFound, ErrDuplicate)
	}

	r.logger.Info("prompt deactivated", "id", p.ID, "name", p.Name, "stage", p.Stage)
	return &p, nil
}
