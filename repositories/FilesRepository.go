package repositories

import (
	"Keyline/database"
	"Keyline/ioc"
	"Keyline/logging"
	"Keyline/middlewares"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/huandu/go-sqlbuilder"
	"time"
)

type File struct {
	id uuid.UUID

	auditCreatedAt time.Time
	auditUpdatedAt time.Time

	name     string
	mimeType string
	content  []byte
}

func NewFile(name string, mimeType string, content []byte) *File {
	return &File{
		name:     name,
		mimeType: mimeType,
		content:  content,
	}
}

func (f *File) Id() uuid.UUID {
	return f.id
}

func (f *File) AuditCreatedAt() time.Time {
	return f.auditCreatedAt
}

func (f *File) AuditUpdatedAt() time.Time {
	return f.auditUpdatedAt
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
	f.id = &id
	return filter
}

type FileRepository struct {
}

func (r *FileRepository) First(ctx context.Context, filter FileFilter) (*File, error) {
	scope := middlewares.GetScope(ctx)
	dbService := ioc.GetDependency[*database.DbService](scope)

	tx, err := dbService.GetTx()
	if err != nil {
		return nil, fmt.Errorf("failed to open tx: %w", err)
	}

	s := sqlbuilder.Select("id", "audit_created_at", "audit_updated_at", "name", "mime_type", "content").
		From("files")

	if filter.id != nil {
		s.Where(s.Equal("id", filter.id))
	}

	s.Limit(1)

	query, args := s.Build()
	logging.Logger.Debug("sql: %s", query)
	row := tx.QueryRow(query, args...)

	var file File
	err = row.Scan(
		&file.id,
		&file.auditCreatedAt,
		&file.auditUpdatedAt,
		&file.name,
		&file.mimeType,
		&file.content,
	)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, nil

	case err != nil:
		return nil, fmt.Errorf("scanning row: %w", err)
	}

	return &file, nil
}

func (r *FileRepository) Insert(ctx context.Context, file *File) error {
	scope := middlewares.GetScope(ctx)
	dbService := ioc.GetDependency[*database.DbService](scope)

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
		).Returning("id", "audit_created_at", "audit_updated_at")

	query, args := s.Build()
	logging.Logger.Debug("sql: %s", query)
	row := tx.QueryRow(query, args...)

	err = row.Scan(&file.id, &file.auditCreatedAt, &file.auditUpdatedAt)
	if err != nil {
		return fmt.Errorf("scanning row: %w", err)
	}

	return nil
}
