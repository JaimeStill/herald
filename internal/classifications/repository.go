package classifications

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"slices"

	"github.com/google/uuid"

	"github.com/JaimeStill/herald/internal/documents"
	"github.com/JaimeStill/herald/internal/prompts"
	"github.com/JaimeStill/herald/internal/workflow"
	"github.com/JaimeStill/herald/pkg/pagination"
	"github.com/JaimeStill/herald/pkg/query"
	"github.com/JaimeStill/herald/pkg/repository"
	"github.com/JaimeStill/herald/pkg/storage"

	gaconfig "github.com/JaimeStill/go-agents/pkg/config"
)

type repo struct {
	db         *sql.DB
	rt         *workflow.Runtime
	logger     *slog.Logger
	pagination pagination.Config
}

// New creates a classification repository implementing the System interface.
// It internally constructs the workflow runtime from the provided dependencies.
func New(
	db *sql.DB,
	agent gaconfig.AgentConfig,
	logger *slog.Logger,
	pagination pagination.Config,
	storage storage.System,
	docs documents.System,
	prompts prompts.System,
) System {
	rt := &workflow.Runtime{
		Agent:     agent,
		Storage:   storage,
		Documents: docs,
		Prompts:   prompts,
		Logger:    logger.With("workflow", "classify"),
	}
	return &repo{
		db:         db,
		rt:         rt,
		logger:     logger.With("system", "classifications"),
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
) (*pagination.PageResult[Classification], error) {
	page.Normalize(r.pagination)

	qb := query.
		NewBuilder(projection, defaultSort).
		WhereSearch(page.Search, "Classification", "Rationale")

	filters.Apply(qb)

	if len(page.Sort) > 0 {
		qb.OrderByFields(page.Sort)
	}

	countSQL, countArgs := qb.BuildCount()
	var total int
	if err := r.db.QueryRowContext(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count classifications: %w", err)
	}

	pageSQL, pageArgs := qb.BuildPage(page.Page, page.PageSize)
	items, err := repository.QueryMany(ctx, r.db, pageSQL, pageArgs, scanClassification)
	if err != nil {
		return nil, fmt.Errorf("query classifications: %w", err)
	}

	result := pagination.NewPageResult(items, total, page.Page, page.PageSize)
	return &result, nil
}

func (r *repo) Find(ctx context.Context, id uuid.UUID) (*Classification, error) {
	q, args := query.NewBuilder(projection).BuildSingle("ID", id)

	c, err := repository.QueryOne(ctx, r.db, q, args, scanClassification)
	if err != nil {
		return nil, repository.MapError(err, ErrNotFound, ErrDuplicate)
	}
	return &c, nil
}

func (r *repo) FindByDocument(ctx context.Context, documentID uuid.UUID) (*Classification, error) {
	q, args := query.NewBuilder(projection).BuildSingle("DocumentID", documentID)

	c, err := repository.QueryOne(ctx, r.db, q, args, scanClassification)
	if err != nil {
		return nil, repository.MapError(err, ErrNotFound, ErrDuplicate)
	}
	return &c, nil
}

func (r *repo) Classify(ctx context.Context, documentID uuid.UUID) (*Classification, error) {
	result, err := workflow.Execute(ctx, r.rt, documentID)
	if err != nil {
		return nil, fmt.Errorf("classify document %s: %w", documentID, err)
	}

	markings := collectMarkings(result.State.Pages)
	markingsJSON, err := json.Marshal(markings)
	if err != nil {
		return nil, fmt.Errorf("marshal markings: %w", err)
	}

	upsertQ := `
		INSERT INTO classifications(
			document_id, classification, confidence, markings_found,
			rationale, model_name, provider_name
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (document_id) DO UPDATE SET
			classification = EXCLUDED.classification,
			confidence = EXCLUDED.confidence,
			markings_found = EXCLUDED.markings_found,
			rationale = EXCLUDED.rationale,
			classified_at = NOW(),
			model_name = EXCLUDED.model_name,
			provider_name = EXCLUDED.provider_name,
			validated_by = NULL,
			validated_at = NULL
		RETURNING id, document_id, classification, confidence, markings_found,
				  rationale, classified_at, model_name, provider_name,
				  validated_by, validated_at`

	upsertArgs := []any{
		documentID,
		result.State.Classification,
		string(result.State.Confidence),
		markingsJSON,
		result.State.Rationale,
		r.rt.Agent.Model.Name,
		r.rt.Agent.Provider.Name,
	}

	c, err := repository.WithTx(ctx, r.db, func(tx *sql.Tx) (Classification, error) {
		cl, err := repository.QueryOne(ctx, tx, upsertQ, upsertArgs, scanClassification)
		if err != nil {
			return Classification{}, fmt.Errorf("upsert classification: %w", err)
		}

		if err := repository.ExecExpectOne(
			ctx, tx,
			"UPDATE documents SET status = 'review', updated_at = NOW() WHERE id = $1",
			documentID,
		); err != nil {
			return Classification{}, fmt.Errorf("update document status: %w", err)
		}

		return cl, nil
	})

	if err != nil {
		return nil, repository.MapError(err, ErrNotFound, ErrDuplicate)
	}

	r.logger.Info("document classified",
		"id", c.ID,
		"document_id", documentID,
		"classification", c.Classification,
		"confidence", c.Confidence,
	)
	return &c, nil
}

func (r *repo) Validate(ctx context.Context, id uuid.UUID, cmd ValidateCommand) (*Classification, error) {
	validateQ := `
		UPDATE classifications
		SET validated_by = $1, validated_at = NOW()
		WHERE id = $2
		RETURNING id, document_id, classification, confidence, markings_found,
				  rationale, classified_at, model_name, provider_name,
				  validated_by, validated_at`

	c, err := repository.WithTx(ctx, r.db, func(tx *sql.Tx) (Classification, error) {
		cl, err := repository.QueryOne(ctx, tx, validateQ, []any{cmd.ValidatedBy, id}, scanClassification)
		if err != nil {
			return Classification{}, repository.MapError(err, ErrNotFound, ErrDuplicate)
		}

		if err := repository.ExecExpectOne(
			ctx, tx,
			"UPDATE documents SET status = 'complete', updated_at = NOW() WHERE id = $1 AND status = 'review'",
			cl.DocumentID,
		); err != nil {
			return Classification{}, ErrInvalidStatus
		}

		return cl, nil
	})

	if err != nil {
		return nil, err
	}

	r.logger.Info("classification validated",
		"id", c.ID,
		"validated_by", c.ValidatedBy,
	)
	return &c, nil
}

func (r *repo) Update(ctx context.Context, id uuid.UUID, cmd UpdateCommand) (*Classification, error) {
	updateQ := `
		UPDATE classifications
		SET classification = $1, rationale = $2, validated_by = $3, validated_at = NOW()
		WHERE id = $4
		RETURNING id, document_id, classification, confidence, markings_found,
				  rationale, classified_at, model_name, provider_name,
				  validated_by, validated_at`

	c, err := repository.WithTx(ctx, r.db, func(tx *sql.Tx) (Classification, error) {
		cl, err := repository.QueryOne(ctx, tx, updateQ,
			[]any{cmd.Classification, cmd.Rationale, cmd.UpdatedBy, id},
			scanClassification,
		)
		if err != nil {
			return Classification{}, repository.MapError(err, ErrNotFound, ErrDuplicate)
		}

		if err := repository.ExecExpectOne(
			ctx, tx,
			"UPDATE documents SET status = 'complete', updated_at = NOW() WHERE id = $1 AND status = 'review'",
			cl.DocumentID,
		); err != nil {
			return Classification{}, ErrInvalidStatus
		}

		return cl, nil
	})

	if err != nil {
		return nil, err
	}

	r.logger.Info("classification updated",
		"id", c.ID,
		"updated_by", cmd.UpdatedBy,
	)
	return &c, nil
}

func (r *repo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := repository.WithTx(ctx, r.db, func(tx *sql.Tx) (struct{}, error) {
		if err := repository.ExecExpectOne(
			ctx, tx,
			"DELETE FROM classifications WHERE id = $1",
			id,
		); err != nil {
			return struct{}{}, err
		}
		return struct{}{}, nil
	})

	if err != nil {
		return repository.MapError(err, ErrNotFound, ErrDuplicate)
	}

	r.logger.Info("classification deleted", "id", id)
	return nil
}

func collectMarkings(pages []workflow.ClassificationPage) []string {
	var all []string
	for _, p := range pages {
		all = append(all, p.MarkingsFound...)
	}

	slices.Sort(all)
	all = slices.Compact(all)

	if all == nil {
		all = []string{}
	}

	return all
}
