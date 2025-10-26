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

type resourceServerRepository struct{}

func NewResourceServerRepository() repositories.ResourceServerRepository {
	return &resourceServerRepository{}
}

func (r *resourceServerRepository) selectQuery(filter repositories.ResourceServerFilter) *sqlbuilder.SelectBuilder {
	s := sqlbuilder.Select(
		"id",
		"audit_created_at",
		"audit_updated_at",
		"version",
		"virtual_server_id",
		"project_id",
		"name",
		"description",
	).From("resource_servers")

	if filter.HasVirtualServerId() {
		s.Where(s.Equal("virtual_server_id", filter.GetVirtualServerId()))
	}

	if filter.HasProjectId() {
		s.Where(s.Equal("project_id", filter.GetProjectId()))
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

func (r *resourceServerRepository) List(ctx context.Context, filter repositories.ResourceServerFilter) ([]*repositories.ResourceServer, int, error) {
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

	var resourceServers []*repositories.ResourceServer
	var totalCount int
	for rows.Next() {
		resourceServer := repositories.ResourceServer{
			ModelBase: repositories.NewModelBase(),
		}
		err = rows.Scan(append(resourceServer.GetScanPointers(), &totalCount)...)
		if err != nil {
			return nil, 0, fmt.Errorf("scanning row: %w", err)
		}

		resourceServers = append(resourceServers, &resourceServer)
	}

	return resourceServers, totalCount, nil
}

func (r *resourceServerRepository) First(ctx context.Context, filter repositories.ResourceServerFilter) (*repositories.ResourceServer, error) {
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

	resourceServer := repositories.ResourceServer{
		ModelBase: repositories.NewModelBase(),
	}
	err = row.Scan(resourceServer.GetScanPointers()...)
	if err != nil {
		return nil, fmt.Errorf("scanning row: %w", err)
	}

	return &resourceServer, nil
}

func (r *resourceServerRepository) Single(ctx context.Context, filter repositories.ResourceServerFilter) (*repositories.ResourceServer, error) {
	resourceServer, err := r.First(ctx, filter)
	if err != nil {
		return nil, err
	}
	if resourceServer == nil {
		return nil, utils.ErrResourceServerNotFound
	}
	return resourceServer, nil
}

func (r *resourceServerRepository) Insert(ctx context.Context, resourceServer *repositories.ResourceServer) error {
	scope := middlewares.GetScope(ctx)
	dbService := ioc.GetDependency[database.DbService](scope)

	tx, err := dbService.GetTx()
	if err != nil {
		return fmt.Errorf("failed to open tx: %w", err)
	}

	s := sqlbuilder.InsertInto("resource_servers").
		Cols(
			"virtual_server_id",
			"project_id",
			"name",
			"description",
		).
		Values(
			resourceServer.VirtualServerId(),
			resourceServer.ProjectId(),
			resourceServer.Name(),
			resourceServer.Description(),
		).Returning("id", "audit_created_at", "audit_updated_at", "version")

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	row := tx.QueryRowContext(ctx, query, args...)
	err = row.Scan(resourceServer.InsertPointers()...)
	if err != nil {
		return fmt.Errorf("scanning row: %w", err)
	}

	resourceServer.ClearChanges()
	return nil
}

func (r *resourceServerRepository) Update(ctx context.Context, resourceServer *repositories.ResourceServer) error {
	scope := middlewares.GetScope(ctx)
	dbService := ioc.GetDependency[database.DbService](scope)

	tx, err := dbService.GetTx()
	if err != nil {
		return fmt.Errorf("failed to open tx: %w", err)
	}

	s := sqlbuilder.Update("resource_servers")
	for fieldName, value := range resourceServer.Changes() {
		s.SetMore(s.Assign(fieldName, value))
	}
	s.SetMore(s.Assign("version", resourceServer.Version()+1))

	s.Where(s.Equal("id", resourceServer.Id()))
	s.Where(s.Equal("version", resourceServer.Version()))
	s.Returning("audit_updated_at", "version")

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	row := tx.QueryRowContext(ctx, query, args...)

	err = row.Scan(resourceServer.UpdatePointers()...)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return fmt.Errorf("updating resource server: %w", repositories.ErrVersionMismatch)

	case err != nil:
		return fmt.Errorf("scanning row: %w", err)
	}

	resourceServer.ClearChanges()
	return nil
}

func (r *resourceServerRepository) Delete(ctx context.Context, id uuid.UUID) error {
	scope := middlewares.GetScope(ctx)
	dbService := ioc.GetDependency[database.DbService](scope)

	tx, err := dbService.GetTx()
	if err != nil {
		return fmt.Errorf("failed to open tx: %w", err)
	}

	s := sqlbuilder.DeleteFrom("resource_servers")
	s.Where(s.Equal("id", id))

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	_, err = tx.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("executing sql: %w", err)
	}

	return nil
}
