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

	"github.com/google/uuid"
	"github.com/huandu/go-sqlbuilder"
	"github.com/lib/pq"
)

type applicationRepository struct{}

func NewApplicationRepository() repositories.ApplicationRepository {
	return &applicationRepository{}
}

func (r *applicationRepository) selectQuery(filter repositories.ApplicationFilter) *sqlbuilder.SelectBuilder {
	s := sqlbuilder.Select(
		"id",
		"audit_created_at",
		"audit_updated_at",
		"version",
		"virtual_server_id",
		"name",
		"display_name",
		"type",
		"hashed_secret",
		"redirect_uris",
		"post_logout_redirect_uris",
		"system_application",
		"claims_mapping_script",
	).From("applications")

	if filter.HasName() {
		s.Where(s.Equal("name", filter.GetName()))
	}

	if filter.HasId() {
		s.Where(s.Equal("id", filter.GetId()))
	}

	if filter.HasIds() {
		s.Where(s.Any("id", "=", pq.Array(filter.GetIds())))
	}

	if filter.HasVirtualServerId() {
		s.Where(s.Equal("virtual_server_id", filter.GetVirtualServerId()))
	}

	if filter.HasSearch() {
		term := filter.GetSearch().Term()
		s.Where(s.Or(
			s.ILike("name", term),
			s.ILike("display_name", term),
		))
	}

	if filter.HasOrder() {
		filter.GetOrderInfo().Apply(s)
	}

	if filter.HasPagination() {
		filter.GetPagingInfo().Apply(s)
	}

	return s
}

func (r *applicationRepository) Update(ctx context.Context, application *repositories.Application) error {
	scope := middlewares.GetScope(ctx)
	dbService := ioc.GetDependency[database.DbService](scope)

	tx, err := dbService.GetTx()
	if err != nil {
		return fmt.Errorf("failed to open tx: %w", err)
	}

	s := sqlbuilder.Update("applications")
	for fieldName, value := range application.Changes() {
		s.SetMore(s.Assign(fieldName, value))
	}
	s.SetMore(s.Assign("version", application.Version()+1))

	s.Where(s.Equal("id", application.Id()))
	s.Where(s.Equal("version", application.Version()))
	s.Returning("audit_updated_at", "version")

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	row := tx.QueryRowContext(ctx, query, args...)

	err = row.Scan(application.UpdatePointers()...)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return fmt.Errorf("updating application: %w", repositories.ErrVersionMismatch)
	case err != nil:
		return fmt.Errorf("scanning row: %w", err)
	}

	application.ClearChanges()
	return nil
}

func (r *applicationRepository) Single(ctx context.Context, filter repositories.ApplicationFilter) (*repositories.Application, error) {
	application, err := r.First(ctx, filter)
	if err != nil {
		return nil, err
	}
	if application == nil {
		return nil, utils.ErrApplicationNotFound
	}
	return application, nil
}

func (r *applicationRepository) First(ctx context.Context, filter repositories.ApplicationFilter) (*repositories.Application, error) {
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

	application := repositories.Application{
		ModelBase: repositories.NewModelBase(),
	}
	err = row.Scan(application.GetScanPointers()...)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, nil

	case err != nil:
		return nil, fmt.Errorf("scanning row: %w", err)
	}

	return &application, nil
}

func (r *applicationRepository) Insert(ctx context.Context, application *repositories.Application) error {
	scope := middlewares.GetScope(ctx)
	dbService := ioc.GetDependency[database.DbService](scope)

	tx, err := dbService.GetTx()
	if err != nil {
		return fmt.Errorf("failed to open tx: %w", err)
	}

	s := sqlbuilder.InsertInto("applications").
		Cols("virtual_server_id", "name", "display_name", "type", "hashed_secret", "redirect_uris", "post_logout_redirect_uris", "system_application", "claims_mapping_script").
		Values(
			application.VirtualServerId(),
			application.Name(),
			application.DisplayName(),
			application.Type(),
			application.HashedSecret(),
			pq.Array(application.RedirectUris()),
			pq.Array(application.PostLogoutRedirectUris()),
			application.SystemApplication(),
			application.ClaimsMappingScript(),
		).Returning("id", "audit_created_at", "audit_updated_at", "version")

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	row := tx.QueryRowContext(ctx, query, args...)

	err = row.Scan(application.InsertPointers()...)
	if err != nil {
		return fmt.Errorf("scanning row: %w", err)
	}

	application.ClearChanges()
	return nil
}

func (r *applicationRepository) List(ctx context.Context, filter repositories.ApplicationFilter) ([]*repositories.Application, int, error) {
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

	var applications []*repositories.Application
	var totalCount int
	for rows.Next() {
		application := repositories.Application{
			ModelBase: repositories.NewModelBase(),
		}

		err = rows.Scan(append(application.GetScanPointers(), &totalCount)...)
		if err != nil {
			return nil, 0, fmt.Errorf("scanning row: %w", err)
		}

		applications = append(applications, &application)
	}

	return applications, totalCount, nil
}

func (r *applicationRepository) Delete(ctx context.Context, id uuid.UUID) error {
	scope := middlewares.GetScope(ctx)
	dbService := ioc.GetDependency[database.DbService](scope)

	tx, err := dbService.GetTx()
	if err != nil {
		return fmt.Errorf("failed to open tx: %w", err)
	}

	s := sqlbuilder.DeleteFrom("applications")

	s.Where(s.Equal("id", id))

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	_, err = tx.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("executing delete: %w", err)
	}

	return nil
}
