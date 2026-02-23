package documents

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/url"
	"path/filepath"

	"github.com/google/uuid"

	"github.com/JaimeStill/herald/pkg/pagination"
	"github.com/JaimeStill/herald/pkg/query"
	"github.com/JaimeStill/herald/pkg/repository"
	"github.com/JaimeStill/herald/pkg/storage"
)

type repo struct {
	db         *sql.DB
	storage    storage.System
	logger     *slog.Logger
	pagination pagination.Config
}

// New creates a document repository implementing the System interface.
func New(
	db *sql.DB,
	store storage.System,
	logger *slog.Logger,
	pagination pagination.Config,
) System {
	return &repo{
		db:         db,
		storage:    store,
		logger:     logger.With("system", "documents"),
		pagination: pagination,
	}
}

func (r *repo) Handler(maxUploadSize int64) *Handler {
	return NewHandler(r, r.logger, r.pagination, maxUploadSize)
}

func (r *repo) List(
	ctx context.Context,
	page pagination.PageRequest,
	filters Filters,
) (*pagination.PageResult[Document], error) {
	page.Normalize(r.pagination)

	qb := query.
		NewBuilder(projection, defaultSort).
		WhereSearch(page.Search, "Filename", "ExternalPlatform")

	filters.Apply(qb)

	if len(page.Sort) > 0 {
		qb.OrderByFields(page.Sort)
	}

	countSQL, countArgs := qb.BuildCount()
	var total int
	if err := r.db.QueryRowContext(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count documents: %w", err)
	}

	pageSQL, pageArgs := qb.BuildPage(page.Page, page.PageSize)
	docs, err := repository.QueryMany(ctx, r.db, pageSQL, pageArgs, scanDocument)
	if err != nil {
		return nil, fmt.Errorf("query documents: %w", err)
	}

	result := pagination.NewPageResult(docs, total, page.Page, page.PageSize)
	return &result, nil
}

func (r repo) Find(ctx context.Context, id uuid.UUID) (*Document, error) {
	q, args := query.NewBuilder(projection).BuildSingle("ID", id)

	d, err := repository.QueryOne(ctx, r.db, q, args, scanDocument)
	if err != nil {
		return nil, repository.MapError(err, ErrNotFound, ErrDuplicate)
	}
	return &d, nil
}

func (r *repo) Create(ctx context.Context, cmd CreateCommand) (*Document, error) {
	id := uuid.New()
	key := buildStorageKey(id, sanitizeFilename(cmd.Filename))

	if err := r.storage.Upload(ctx, key, bytes.NewReader(cmd.Data), cmd.ContentType); err != nil {
		return nil, fmt.Errorf("upload document blob: %w", err)
	}

	q := `
		INSERT INTO documents(id, external_id, external_platform, filename, content_type, size_bytes, page_count, storage_key)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, external_id, external_platform, filename, content_type, size_bytes, page_count, storage_key, status, uploaded_at, updated_at`

	insertArgs := []any{
		id,
		cmd.ExternalID,
		cmd.ExternalPlatform,
		cmd.Filename,
		cmd.ContentType,
		int64(len(cmd.Data)),
		cmd.PageCount,
		key,
	}

	d, err := repository.WithTx(ctx, r.db, func(tx *sql.Tx) (Document, error) {
		return repository.QueryOne(ctx, tx, q, insertArgs, scanDocument)
	})

	if err != nil {
		if delErr := r.storage.Delete(ctx, key); delErr != nil {
			r.logger.Warn("compensating blob delete failed", "key", key, "error", delErr)
		}
		return nil, repository.MapError(err, ErrNotFound, ErrDuplicate)
	}

	r.logger.Info("document created", "id", d.ID, "filename", d.Filename)
	return &d, nil
}

func (r *repo) Delete(ctx context.Context, id uuid.UUID) error {
	doc, err := r.Find(ctx, id)
	if err != nil {
		return err
	}

	_, err = repository.WithTx(ctx, r.db, func(tx *sql.Tx) (struct{}, error) {
		if err := repository.ExecExpectOne(
			ctx, tx,
			"DELETE FROM documents WHERE id = $1",
			id,
		); err != nil {
			return struct{}{}, err
		}
		return struct{}{}, nil
	})

	if err != nil {
		return repository.MapError(err, ErrNotFound, ErrDuplicate)
	}

	if delErr := r.storage.Delete(ctx, doc.StorageKey); delErr != nil {
		r.logger.Warn(
			"blob delete failed after DB delete",
			"key", doc.StorageKey,
			"error", delErr,
		)
	}

	r.logger.Info("document deleted", "id", id)
	return nil
}

func buildStorageKey(id uuid.UUID, filename string) string {
	return fmt.Sprintf("documents/%s/%s", id, filename)
}

func sanitizeFilename(name string) string {
	name = filepath.Base(name)
	if name == "." || name == "" {
		name = "document"
	}
	return url.PathEscape(name)
}
