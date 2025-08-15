package repositories

import (
	"Keyline/database"
	"Keyline/ioc"
	"Keyline/logging"
	"Keyline/middlewares"
	"context"
	"fmt"
	"github.com/google/uuid"
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
	id uuid.UUID
}

func NewFileFilter() FileFilter {
	return FileFilter{}
}

func (f FileFilter) Clone() FileFilter {
	return f
}

func (f FileFilter) Id(id uuid.UUID) FileFilter {
	filter := f.Clone()
	f.id = id
	return filter
}

type FileRepository struct {
}

func (r *FileRepository) Insert(ctx context.Context, file *File) error {
	scope := middlewares.GetScope(ctx)
	dbService := ioc.GetDependency[*database.DbService](scope)

	tx, err := dbService.GetTx()
	if err != nil {
		return fmt.Errorf("failed to open tx: %w", err)
	}

	s := `
insert into files 
	(name, mime_type, content)
values ($1, $2, $3)
returning id, audit_created_at, audit_updated_at`

	logging.Logger.Debug("sql: %s", s)
	row := tx.QueryRow(
		s,
		file.name,
		file.mimeType,
		file.content,
	)

	err = row.Scan(&file.id, &file.auditCreatedAt, &file.auditUpdatedAt)
	if err != nil {
		return fmt.Errorf("scanning row: %w", err)
	}

	return nil
}
