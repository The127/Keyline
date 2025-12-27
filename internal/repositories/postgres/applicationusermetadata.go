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

	"github.com/google/uuid"

	"github.com/huandu/go-sqlbuilder"
)

type postgresApplicationUserMetadata struct {
	postgresBaseModel
	applicationId uuid.UUID
	userId        uuid.UUID
	metadata      string
}

func mapApplicationUserMetadata(m *repositories.ApplicationUserMetadata) *postgresApplicationUserMetadata {
	return &postgresApplicationUserMetadata{
		postgresBaseModel: mapBase(m.BaseModel),
		applicationId:     m.ApplicationId(),
		userId:            m.UserId(),
		metadata:          m.Metadata(),
	}
}

func (m *postgresApplicationUserMetadata) Map() *repositories.ApplicationUserMetadata {
	return repositories.NewApplicationUserMetadataFromDB(
		m.MapBase(),
		m.applicationId,
		m.userId,
		m.metadata,
	)
}

func (m *postgresApplicationUserMetadata) scan(row pghelpers.Row, additionalPtrs ...any) error {
	ptrs := []any{
		&m.id,
		&m.auditCreatedAt,
		&m.auditUpdatedAt,
		&m.xmin,
		&m.applicationId,
		&m.userId,
		&m.metadata,
	}

	ptrs = append(ptrs, additionalPtrs...)

	return row.Scan(ptrs...)
}

type ApplicationUserMetadataRepository struct {
	db            *sql.DB
	changeTracker *change.Tracker
	entityType    int
}

func NewApplicationUserMetadataRepository(db *sql.DB, changeTracker *change.Tracker, entityType int) *ApplicationUserMetadataRepository {
	return &ApplicationUserMetadataRepository{
		db:            db,
		changeTracker: changeTracker,
		entityType:    entityType,
	}
}

func (r *ApplicationUserMetadataRepository) selectQuery(filter *repositories.ApplicationUserMetadataFilter) *sqlbuilder.SelectBuilder {
	s := sqlbuilder.Select(
		"id",
		"audit_created_at",
		"audit_updated_at",
		"xmin",
		"application_id",
		"user_id",
		"metadata",
	).From("application_user_metadata")

	if filter.HasApplicationId() {
		s.Where(s.Equal("application_id", filter.GetApplicationId()))
	}

	if filter.HasApplicationIds() {
		s.Where(s.In("application_id", filter.GetApplicationIds()))
	}

	if filter.HasUserId() {
		s.Where(s.Equal("user_id", filter.GetUserId()))
	}

	return s
}

func (r *ApplicationUserMetadataRepository) List(ctx context.Context, filter *repositories.ApplicationUserMetadataFilter) ([]*repositories.ApplicationUserMetadata, int, error) {
	s := r.selectQuery(filter)
	s.SelectMore("count(*) over()")

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("querying db: %w", err)
	}
	defer utils.PanicOnError(rows.Close, "closing rows")

	var metadata []*repositories.ApplicationUserMetadata
	var totalCount int
	for rows.Next() {
		m := &postgresApplicationUserMetadata{}
		err := m.scan(rows, &totalCount)
		if err != nil {
			return nil, 0, fmt.Errorf("scanning row: %w", err)
		}

		metadata = append(metadata, m.Map())
	}

	return metadata, totalCount, nil
}

func (r *ApplicationUserMetadataRepository) Single(ctx context.Context, filter *repositories.ApplicationUserMetadataFilter) (*repositories.ApplicationUserMetadata, error) {
	result, err := r.FirstOrNil(ctx, filter)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, utils.ErrUserApplicationMetadataNotFound
	}
	return result, nil
}

func (r *ApplicationUserMetadataRepository) FirstOrNil(ctx context.Context, filter *repositories.ApplicationUserMetadataFilter) (*repositories.ApplicationUserMetadata, error) {
	s := r.selectQuery(filter)
	s.Limit(1)

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	row := r.db.QueryRowContext(ctx, query, args...)

	metadata := &postgresApplicationUserMetadata{}
	err := metadata.scan(row)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, nil
	case err != nil:
		return nil, fmt.Errorf("scanning row: %w", err)
	}

	return metadata.Map(), nil
}

func (r *ApplicationUserMetadataRepository) Insert(applicationUserMetadata *repositories.ApplicationUserMetadata) {
	r.changeTracker.Add(change.NewEntry(change.Added, r.entityType, applicationUserMetadata))
}

func (r *ApplicationUserMetadataRepository) ExecuteInsert(ctx context.Context, tx *sql.Tx, applicationUserMetadata *repositories.ApplicationUserMetadata) error {
	mapped := mapApplicationUserMetadata(applicationUserMetadata)

	s := sqlbuilder.InsertInto("application_user_metadata").
		Cols(
			"id",
			"audit_created_at",
			"audit_updated_at",
			"application_id",
			"user_id",
			"metadata",
		).
		Values(
			mapped.id,
			mapped.auditCreatedAt,
			mapped.auditUpdatedAt,
			mapped.applicationId,
			mapped.userId,
			mapped.metadata,
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

	applicationUserMetadata.SetVersion(xmin)
	applicationUserMetadata.ClearChanges()
	return nil
}

func (r *ApplicationUserMetadataRepository) Update(applicationUserMetadata *repositories.ApplicationUserMetadata) {
	r.changeTracker.Add(change.NewEntry(change.Updated, r.entityType, applicationUserMetadata))
}

func (r *ApplicationUserMetadataRepository) ExecuteUpdate(ctx context.Context, tx *sql.Tx, applicationUserMetadata *repositories.ApplicationUserMetadata) error {
	if !applicationUserMetadata.HasChanges() {
		return nil
	}

	mapped := mapApplicationUserMetadata(applicationUserMetadata)

	s := sqlbuilder.Update("application_user_metadata")
	s.Where(s.Equal("id", mapped.id))
	s.Where(s.Equal("xmin", mapped.xmin))

	for _, field := range applicationUserMetadata.GetChanges() {
		switch field {
		case repositories.ApplicationUserMetadataChangeMetadata:
			s.SetMore(s.Assign("metadata", mapped.metadata))

		default:
			return fmt.Errorf("updating field %v is not supported", field)
		}
	}

	s.Returning("xmin")
	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	row := tx.QueryRowContext(ctx, query, args...)

	var xmin uint32
	err := row.Scan(&xmin)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return fmt.Errorf("updating application: %w", repositories.ErrVersionMismatch)
	case err != nil:
		return fmt.Errorf("scanning row: %w", err)
	}

	applicationUserMetadata.SetVersion(xmin)
	applicationUserMetadata.ClearChanges()
	return nil
}
