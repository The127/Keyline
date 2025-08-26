package repositories

import (
	"Keyline/database"
	"Keyline/ioc"
	"Keyline/logging"
	"Keyline/middlewares"
	"Keyline/utils"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/huandu/go-sqlbuilder"
)

type File struct {
	ModelBase

	name     string
	mimeType string
	content  []byte
}

func NewFile(name string, mimeType string, content []byte) *File {
	return &File{
		ModelBase: NewModelBase(),
		name:      name,
		mimeType:  mimeType,
		content:   content,
	}
}

func (f *File) getScanPointers() []any {
	return []any{
		&f.id,
		&f.auditCreatedAt,
		&f.auditUpdatedAt,
		&f.version,
		&f.name,
		&f.mimeType,
		&f.content,
	}
}

func (f *File) Name() string {
	return f.name
}

func (f *File) MimeType() string {
	return f.mimeType
}

func (f *File) Content() []byte {
	return f.content
}

type FileFilter struct {
	id *uuid.UUID
}

func NewFileFilter() FileFilter {
	return FileFilter{}
}

func (f FileFilter) Clone() FileFilter {
	return f
}

func (f FileFilter) Id(id uuid.UUID) FileFilter {
	filter := f.Clone()
	filter.id = &id
	return filter
}

type FileRepository interface {
	Single(ctx context.Context, filter FileFilter) (*File, error)
	First(ctx context.Context, filter FileFilter) (*File, error)
	Insert(ctx context.Context, file *File) error
}

type fileRepository struct {
}

func NewFileRepository() FileRepository {
	return &fileRepository{}
}

func (r *fileRepository) selectQuery(filter FileFilter) *sqlbuilder.SelectBuilder {
	s := sqlbuilder.Select(
		"id",
		"audit_created_at",
		"audit_updated_at",
		"version",
		"name",
		"mime_type",
		"content",
	).From("files")

	if filter.id != nil {
		s.Where(s.Equal("id", filter.id))
	}

	return s
}

func (r *fileRepository) Single(ctx context.Context, filter FileFilter) (*File, error) {
	file, err := r.First(ctx, filter)
	if err != nil {
		return nil, err
	}
	if file == nil {
		return nil, utils.ErrFileNotFoud
	}
	return file, nil
}

func (r *fileRepository) First(ctx context.Context, filter FileFilter) (*File, error) {
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

	file := File{
		ModelBase: NewModelBase(),
	}
	err = row.Scan(file.getScanPointers()...)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, nil

	case err != nil:
		return nil, fmt.Errorf("scanning row: %w", err)
	}

	return &file, nil
}

func (r *fileRepository) Insert(ctx context.Context, file *File) error {
	scope := middlewares.GetScope(ctx)
	dbService := ioc.GetDependency[database.DbService](scope)

	tx, err := dbService.GetTx()
	if err != nil {
		return fmt.Errorf("failed to open tx: %w", err)
	}

	s := sqlbuilder.InsertInto("files").
		Cols("name", "mime_type", "content").
		Values(
			file.name,
			file.mimeType,
			file.content,
		).Returning("id", "audit_created_at", "audit_updated_at", "version")

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	row := tx.QueryRowContext(ctx, query, args...)

	err = row.Scan(&file.id, &file.auditCreatedAt, &file.auditUpdatedAt, &file.version)
	if err != nil {
		return fmt.Errorf("scanning row: %w", err)
	}

	file.clearChanges()
	return nil
}
