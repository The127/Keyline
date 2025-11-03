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
)

type resourceServerScopeRepository struct{}

func NewResourceServerScopeRepository() repositories.ResourceServerScopeRepository {
	return &resourceServerScopeRepository{}
}

func (r *resourceServerScopeRepository) selectQuery(filter repositories.ResourceServerScopeFilter) *sqlbuilder.SelectBuilder {
	s := sqlbuilder.Select(
		"id",
		"audit_created_at",
		"audit_updated_at",
		"audit_version",
		"virtual_server_id",
		"project_id",
		"resource_server_id",
		"scope",
		"name",
		"description",
	).From("resource_server_scopes")

	if filter.HasVirtualServerId() {
		s.Where(s.Equal("virtual_server_id", filter.GetVirtualServerId()))
	}

	if filter.HasProjectId() {
		s.Where(s.Equal("project_id", filter.GetProjectId()))
	}

	if filter.HasResourceServerId() {
		s.Where(s.Equal("resource_server_id", filter.GetResourceServerId()))
	}

	if filter.HasId() {
		s.Where(s.Equal("id", filter.GetId()))
	}

	if filter.HasSearch() {
		term := filter.GetSearch().Term()
		s.Where(s.Or(
			s.ILike("name", term),
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

func (r *resourceServerScopeRepository) List(ctx context.Context, filter repositories.ResourceServerScopeFilter) ([]*repositories.ResourceServerScope, int, error) {
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

	var resourceServerScopes []*repositories.ResourceServerScope
	var totalCount int
	for rows.Next() {
		resourceServerScope := repositories.ResourceServerScope{
			ModelBase: repositories.NewModelBase(),
		}
		err = rows.Scan(append(resourceServerScope.GetScanPointers(), &totalCount)...)
		if err != nil {
			return nil, 0, fmt.Errorf("scanning row: %w", err)
		}
		resourceServerScopes = append(resourceServerScopes, &resourceServerScope)
	}

	return resourceServerScopes, totalCount, nil
}

func (r *resourceServerScopeRepository) First(ctx context.Context, filter repositories.ResourceServerScopeFilter) (*repositories.ResourceServerScope, error) {
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

	resourceServerScope := repositories.ResourceServerScope{
		ModelBase: repositories.NewModelBase(),
	}
	err = row.Scan(resourceServerScope.GetScanPointers()...)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, nil

	case err != nil:
		return nil, fmt.Errorf("scanning row: %w", err)
	}

	return &resourceServerScope, nil
}

func (r *resourceServerScopeRepository) Single(ctx context.Context, filter repositories.ResourceServerScopeFilter) (*repositories.ResourceServerScope, error) {
	resourceServerScope, err := r.First(ctx, filter)
	if err != nil {
		return nil, err
	}
	if resourceServerScope == nil {
		return nil, utils.ErrResourceServerScopeNotFound
	}
	return resourceServerScope, nil
}

func (r *resourceServerScopeRepository) Insert(ctx context.Context, resourceServerScope *repositories.ResourceServerScope) error {
	scope := middlewares.GetScope(ctx)
	dbService := ioc.GetDependency[database.DbService](scope)

	tx, err := dbService.GetTx()
	if err != nil {
		return fmt.Errorf("failed to open tx: %w", err)
	}

	s := sqlbuilder.InsertInto("resource_server_scopes").
		Cols(
			"virtual_server_id",
			"project_id",
			"resource_server_id",
			"scope",
			"name",
			"description",
		).
		Values(
			resourceServerScope.VirtualServerId(),
			resourceServerScope.ProjectId(),
			resourceServerScope.ResourceServerId(),
			resourceServerScope.Scope(),
			resourceServerScope.Name(),
			resourceServerScope.Description(),
		).Returning("id", "audit_created_at", "audit_updated_at", "version")

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	row := tx.QueryRowContext(ctx, query, args...)
	err = row.Scan(resourceServerScope.InsertPointers()...)
	if err != nil {
		return fmt.Errorf("scanning row: %w", err)
	}

	resourceServerScope.ClearChanges()
	return nil
}

func (r *resourceServerScopeRepository) Update(ctx context.Context, resourceServerScope *repositories.ResourceServerScope) error {
	scope := middlewares.GetScope(ctx)
	dbService := ioc.GetDependency[database.DbService](scope)

	tx, err := dbService.GetTx()
	if err != nil {
		return fmt.Errorf("failed to open tx: %w", err)
	}

	s := sqlbuilder.Update("resource_server_scopes")
	for fieldName, value := range resourceServerScope.Changes() {
		s.SetMore(s.Assign(fieldName, value))
	}
	s.SetMore(s.Assign("version", resourceServerScope.Version()+1))

	s.Where(s.Equal("id", resourceServerScope.Id()))
	s.Where(s.Equal("version", resourceServerScope.Version()))
	s.Returning("audit_updated_at", "version")

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	row := tx.QueryRowContext(ctx, query, args...)

	err = row.Scan(resourceServerScope.UpdatePointers()...)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return fmt.Errorf("updating resource server scope: %w", repositories.ErrVersionMismatch)

	case err != nil:
		return fmt.Errorf("scanning row: %w", err)
	}

	resourceServerScope.ClearChanges()
	return nil
}

func (r *resourceServerScopeRepository) Delete(ctx context.Context, id uuid.UUID) error {
	scope := middlewares.GetScope(ctx)
	dbService := ioc.GetDependency[database.DbService](scope)

	tx, err := dbService.GetTx()
	if err != nil {
		return fmt.Errorf("failed to open tx: %w", err)
	}

	s := sqlbuilder.DeleteFrom("resource_server_scopes")
	s.Where(s.Equal("id", id))

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	_, err = tx.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("executing sql: %w", err)
	}

	return nil
}
