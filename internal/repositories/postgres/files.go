package postgres

import (
	"Keyline/internal/database"
	"Keyline/internal/logging"
	"Keyline/internal/middlewares"
	"Keyline/internal/repositories"
	"Keyline/ioc"
	"Keyline/utils"
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/huandu/go-sqlbuilder"
)

type fileRepository struct {
}

func NewFileRepository() repositories.FileRepository {
	return &fileRepository{}
}

func (r *fileRepository) selectQuery(filter repositories.FileFilter) *sqlbuilder.SelectBuilder {
	s := sqlbuilder.Select(
		"id",
		"audit_created_at",
		"audit_updated_at",
		"version",
		"name",
		"mime_type",
		"content",
	).From("files")

	if filter.HasId() {
		s.Where(s.Equal("id", filter.GetId()))
	}

	return s
}

func (r *fileRepository) Single(ctx context.Context, filter repositories.FileFilter) (*repositories.File, error) {
	file, err := r.First(ctx, filter)
	if err != nil {
		return nil, err
	}
	if file == nil {
		return nil, utils.ErrFileNotFoud
	}
	return file, nil
}

func (r *fileRepository) First(ctx context.Context, filter repositories.FileFilter) (*repositories.File, error) {
	scope := middlewares.GetScope(ctx)
	dbService := ioc.GetDependency[database.DbService](scope)

	tx, err := dbService.GetTx()
	if err != nil {
		return nil, fmt.Errorf("failed to open tx: %w", err)
	}

	s := r.selectQuery(filter)

	s.Limit(1)

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	row := tx.QueryRowContext(ctx, query, args...)

	file := repositories.File{
		ModelBase: repositories.NewModelBase(),
	}
	err = row.Scan(file.GetScanPointers()...)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, nil

	case err != nil:
		return nil, fmt.Errorf("scanning row: %w", err)
	}

	return &file, nil
}

func (r *fileRepository) Insert(ctx context.Context, file *repositories.File) error {
	scope := middlewares.GetScope(ctx)
	dbService := ioc.GetDependency[database.DbService](scope)

	tx, err := dbService.GetTx()
	if err != nil {
		return fmt.Errorf("failed to open tx: %w", err)
	}

	s := sqlbuilder.InsertInto("files").
		Cols("name", "mime_type", "content").
		Values(
			file.Name(),
			file.MimeType(),
			file.Content(),
		).Returning("id", "audit_created_at", "audit_updated_at", "version")

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	row := tx.QueryRowContext(ctx, query, args...)

	err = row.Scan(file.InsertPointers()...)
	if err != nil {
		return fmt.Errorf("scanning row: %w", err)
	}

	file.ClearChanges()
	return nil
}
