package postgres

import (
	"Keyline/internal/change"
	"Keyline/internal/logging"
	"Keyline/internal/repositories"
	"Keyline/internal/repositories/postgres/pghelpers"
	"Keyline/utils"
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/huandu/go-sqlbuilder"
)

type postgresFile struct {
	postgresBaseModel
	name     string
	mimeType string
	content  []byte
}

func mapFile(file *repositories.File) *postgresFile {
	return &postgresFile{
		postgresBaseModel: mapBase(file.BaseModel),
		name:              file.Name(),
		mimeType:          file.MimeType(),
		content:           file.Content(),
	}
}

func (f *postgresFile) Map() *repositories.File {
	return repositories.NewFileFromDB(
		f.MapBase(),
		f.name,
		f.mimeType,
		f.content,
	)
}

func (f *postgresFile) scan(row pghelpers.Row) error {
	return row.Scan(
		&f.id,
		&f.auditCreatedAt,
		&f.auditUpdatedAt,
		&f.xmin,
		&f.name,
		&f.mimeType,
		&f.content,
	)
}

type FileRepository struct {
	db            *sql.DB
	changeTracker *change.Tracker
	entityType    int
}

func NewFileRepository(db *sql.DB, changeTracker *change.Tracker, entityType int) *FileRepository {
	return &FileRepository{
		db:            db,
		changeTracker: changeTracker,
		entityType:    entityType,
	}
}

func (r *FileRepository) selectQuery(filter *repositories.FileFilter) *sqlbuilder.SelectBuilder {
	s := sqlbuilder.Select(
		"id",
		"audit_created_at",
		"audit_updated_at",
		"xmin",
		"name",
		"mime_type",
		"content",
	).From("files")

	if filter.HasId() {
		s.Where(s.Equal("id", filter.GetId()))
	}

	return s
}

func (r *FileRepository) Single(ctx context.Context, filter *repositories.FileFilter) (*repositories.File, error) {
	file, err := r.First(ctx, filter)
	if err != nil {
		return nil, err
	}
	if file == nil {
		return nil, utils.ErrFileNotFoud
	}
	return file, nil
}

func (r *FileRepository) First(ctx context.Context, filter *repositories.FileFilter) (*repositories.File, error) {
	s := r.selectQuery(filter)

	s.Limit(1)

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	row := r.db.QueryRowContext(ctx, query, args...)

	file := &postgresFile{}
	err := file.scan(row)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, nil

	case err != nil:
		return nil, fmt.Errorf("scanning row: %w", err)
	}

	return file.Map(), nil
}

func (r *FileRepository) Insert(file *repositories.File) {
	r.changeTracker.Add(change.NewEntry(change.Added, r.entityType, file))
}

func (r *FileRepository) ExecuteInsert(ctx context.Context, tx *sql.Tx, file *repositories.File) error {
	mapped := mapFile(file)

	s := sqlbuilder.InsertInto("files").
		Cols(
			"id",
			"audit_created_at",
			"audit_updated_at",
			"name",
			"mime_type",
			"content",
		).
		Values(
			mapped.id,
			mapped.auditCreatedAt,
			mapped.auditUpdatedAt,
			mapped.name,
			mapped.mimeType,
			mapped.content,
		).
		Returning("xmin")

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	row := tx.QueryRowContext(ctx, query, args...)

	var xmin uint32
	err := row.Scan(&xmin)
	if err != nil {
		return fmt.Errorf("scanning row: %w", err)
	}

	file.SetVersion(xmin)
	return nil
}
