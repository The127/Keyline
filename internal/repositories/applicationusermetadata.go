package repositories

import (
	"Keyline/internal/database"
	"Keyline/internal/logging"
	"Keyline/internal/middlewares"
	"Keyline/ioc"
	"Keyline/utils"
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/huandu/go-sqlbuilder"
)

type ApplicationUserMetadata struct {
	ModelBase

	applicationId uuid.UUID
	userId        uuid.UUID

	metadata string
}

func NewApplicationUserMetadata(applicationId uuid.UUID, userId uuid.UUID, metadata string) *ApplicationUserMetadata {
	return &ApplicationUserMetadata{
		ModelBase:     NewModelBase(),
		applicationId: applicationId,
		userId:        userId,
		metadata:      metadata,
	}
}

func (a *ApplicationUserMetadata) getScanPointers() []any {
	return []any{
		&a.id,
		&a.auditCreatedAt,
		&a.auditUpdatedAt,
		&a.version,
		&a.applicationId,
		&a.userId,
		&a.metadata,
	}
}

func (a *ApplicationUserMetadata) ApplicationId() uuid.UUID {
	return a.applicationId
}

func (a *ApplicationUserMetadata) UserId() uuid.UUID {
	return a.userId
}

func (a *ApplicationUserMetadata) Metadata() string {
	return a.metadata
}

func (a *ApplicationUserMetadata) SetMetadata(metadata string) {
	a.metadata = metadata
	a.TrackChange("metadata", metadata)
}

type ApplicationUserMetadataFilter struct {
	applicationId *[]uuid.UUID
	userId        *uuid.UUID
}

func NewApplicationUserMetadataFilter() ApplicationUserMetadataFilter {
	return ApplicationUserMetadataFilter{}
}

func (f ApplicationUserMetadataFilter) Clone() ApplicationUserMetadataFilter {
	return f
}

func (f ApplicationUserMetadataFilter) ApplicationId(applicationId uuid.UUID) ApplicationUserMetadataFilter {
	filter := f.Clone()
	filter.applicationId = &[]uuid.UUID{applicationId}
	return filter
}

func (f ApplicationUserMetadataFilter) ApplicationIds(applicationIds []uuid.UUID) ApplicationUserMetadataFilter {
	filter := f.Clone()
	filter.applicationId = &applicationIds
	return filter
}

func (f ApplicationUserMetadataFilter) UserId(userId uuid.UUID) ApplicationUserMetadataFilter {
	filter := f.Clone()
	filter.userId = &userId
	return filter
}

//go:generate mockgen -destination=./mocks/application_user_metadata_repository.go -package=mocks Keyline/internal/repositories ApplicationUserMetadataRepository
type ApplicationUserMetadataRepository interface {
	List(ctx context.Context, filter ApplicationUserMetadataFilter) ([]*ApplicationUserMetadata, int, error)
}

type applicationUserMetadataRepository struct{}

func NewApplicationUserMetadataRepository() ApplicationUserMetadataRepository {
	return &applicationUserMetadataRepository{}
}

func (r *applicationUserMetadataRepository) selectQuery(filter ApplicationUserMetadataFilter) *sqlbuilder.SelectBuilder {
	s := sqlbuilder.Select(
		"id",
		"audit_created_at",
		"audit_updated_at",
		"version",
		"application_id",
		"user_id",
		"metadata",
	).From("application_user_metadata")

	if filter.applicationId != nil {
		switch len(*filter.applicationId) {
		case 0:
			s.Where(s.Equal("application_id", uuid.Nil)) // should match no rows
		case 1:
			s.Where(s.Equal("application_id", (*filter.applicationId)[0]))
		default:
			s.Where(s.In("application_id", *filter.applicationId))
		}
	}

	if filter.userId != nil {
		s.Where(s.Equal("user_id", filter.userId))
	}

	return s
}

func (r *applicationUserMetadataRepository) List(ctx context.Context, filter ApplicationUserMetadataFilter) ([]*ApplicationUserMetadata, int, error) {
	scope := middlewares.GetScope(ctx)
	dbService := ioc.GetDependency[database.DbService](scope)

	tx, err := dbService.GetTx()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to open tx: %w", err)
	}

	s := r.selectQuery(filter)
	s.SelectMore("count(*) over()")

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	rows, err := tx.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("querying db: %w", err)
	}
	defer utils.PanicOnError(rows.Close, "closing rows")

	var metadata []*ApplicationUserMetadata
	var totalCount int
	for rows.Next() {
		m := ApplicationUserMetadata{
			ModelBase: NewModelBase(),
		}

		err = rows.Scan(append(m.getScanPointers(), &totalCount)...)
		if err != nil {
			return nil, 0, fmt.Errorf("scanning row: %w", err)
		}

		metadata = append(metadata, &m)
	}

	return metadata, totalCount, nil
}
