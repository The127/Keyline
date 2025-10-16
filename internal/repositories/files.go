package repositories

import (
	"Keyline/utils"
	"context"

	"github.com/google/uuid"
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

func (f *File) GetScanPointers() []any {
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

func (f FileFilter) HasId() bool {
	return f.id != nil
}

func (f FileFilter) GetId() uuid.UUID {
	return utils.ZeroIfNil(f.id)
}

//go:generate mockgen -destination=./mocks/file_repository.go -package=mocks Keyline/internal/repositories FileRepository
type FileRepository interface {
	Single(ctx context.Context, filter FileFilter) (*File, error)
	First(ctx context.Context, filter FileFilter) (*File, error)
	Insert(ctx context.Context, file *File) error
}
