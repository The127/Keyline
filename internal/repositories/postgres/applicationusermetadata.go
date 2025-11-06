package postgres

import (
	"Keyline/internal/database"
	"Keyline/internal/logging"
	"Keyline/internal/middlewares"
	"Keyline/internal/repositories"
	"Keyline/utils"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/The127/ioc"

	"github.com/huandu/go-sqlbuilder"
)

type applicationUserMetadataRepository struct{}

func NewApplicationUserMetadataRepository() repositories.ApplicationUserMetadataRepository {
	return &applicationUserMetadataRepository{}
}

func (r *applicationUserMetadataRepository) selectQuery(filter repositories.ApplicationUserMetadataFilter) *sqlbuilder.SelectBuilder {
	s := sqlbuilder.Select(
		"id",
		"audit_created_at",
		"audit_updated_at",
		"version",
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

func (r *applicationUserMetadataRepository) List(ctx context.Context, filter repositories.ApplicationUserMetadataFilter) ([]*repositories.ApplicationUserMetadata, int, error) {
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

	var metadata []*repositories.ApplicationUserMetadata
	var totalCount int
	for rows.Next() {
		m := repositories.ApplicationUserMetadata{
			ModelBase: repositories.NewModelBase(),
		}

		err = rows.Scan(append(m.GetScanPointers(), &totalCount)...)
		if err != nil {
			return nil, 0, fmt.Errorf("scanning row: %w", err)
		}

		metadata = append(metadata, &m)
	}

	return metadata, totalCount, nil
}

func (r *applicationUserMetadataRepository) Single(ctx context.Context, filter repositories.ApplicationUserMetadataFilter) (*repositories.ApplicationUserMetadata, error) {
	result, err := r.First(ctx, filter)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, utils.ErrUserApplicationMetadataNotFound
	}
	return result, nil
}

func (r *applicationUserMetadataRepository) First(ctx context.Context, filter repositories.ApplicationUserMetadataFilter) (*repositories.ApplicationUserMetadata, error) {
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

	metadata := repositories.ApplicationUserMetadata{
		ModelBase: repositories.NewModelBase(),
	}

	err = row.Scan(metadata.GetScanPointers()...)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, nil
	case err != nil:
		return nil, fmt.Errorf("scanning row: %w", err)
	}

	return &metadata, nil
}

func (r *applicationUserMetadataRepository) Insert(ctx context.Context, applicationUserMetadata *repositories.ApplicationUserMetadata) error {
	scope := middlewares.GetScope(ctx)
	dbService := ioc.GetDependency[database.DbService](scope)

	tx, err := dbService.GetTx()
	if err != nil {
		return fmt.Errorf("failed to open tx: %w", err)
	}

	s := sqlbuilder.InsertInto("application_user_metadata").
		Cols(
			"application_id",
			"user_id",
			"metadata",
		).
		Values(
			applicationUserMetadata.ApplicationId(),
			applicationUserMetadata.UserId(),
			applicationUserMetadata.Metadata(),
		).Returning("id", "audit_created_at", "audit_updated_at", "version")

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	row := tx.QueryRowContext(ctx, query, args...)

	err = row.Scan(applicationUserMetadata.InsertPointers()...)
	if err != nil {
		return fmt.Errorf("scanning row: %w", err)
	}

	return nil
}

func (r *applicationUserMetadataRepository) Update(ctx context.Context, applicationUserMetadata *repositories.ApplicationUserMetadata) error {
	scope := middlewares.GetScope(ctx)
	dbService := ioc.GetDependency[database.DbService](scope)

	tx, err := dbService.GetTx()
	if err != nil {
		return fmt.Errorf("failed to open tx: %w", err)
	}

	s := sqlbuilder.Update("application_user_metadata")
	for fieldName, value := range applicationUserMetadata.Changes() {
		s.SetMore(s.Assign(fieldName, value))
	}
	s.SetMore(s.Assign("version", applicationUserMetadata.Version()+1))

	s.Where(s.Equal("id", applicationUserMetadata.Id()))
	s.Where(s.Equal("version", applicationUserMetadata.Version()))
	s.Returning("audit_updated_at", "version")

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	row := tx.QueryRowContext(ctx, query, args...)

	err = row.Scan(applicationUserMetadata.UpdatePointers()...)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return fmt.Errorf("updating application user metadata: %w", repositories.ErrVersionMismatch)
	case err != nil:
		return fmt.Errorf("scanning row: %w", err)
	}

	applicationUserMetadata.ClearChanges()
	return nil
}
