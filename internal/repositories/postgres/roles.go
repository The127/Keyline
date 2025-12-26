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

	"github.com/google/uuid"
	"github.com/huandu/go-sqlbuilder"
)

type roleRepository struct {
}

func NewRoleRepository() repositories.RoleRepository {
	return &roleRepository{}
}

func (r *roleRepository) selectQuery(filter repositories.RoleFilter) *sqlbuilder.SelectBuilder {
	s := sqlbuilder.Select(
		"id",
		"audit_created_at",
		"audit_updated_at",
		"version",
		"virtual_server_id",
		"project_id",
		"name",
		"description",
	).From("roles")

	if filter.HasName() {
		s.Where(s.Equal("name", filter.GetName()))
	}

	if filter.HasId() {
		s.Where(s.Equal("id", filter.GetId()))
	}

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

func (r *roleRepository) List(ctx context.Context, filter repositories.RoleFilter) ([]*repositories.Role, int, error) {
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
	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("querying rows: %w", err)
	}
	defer utils.PanicOnError(rows.Close, "closing rows")

	var roles []*repositories.Role
	var totalCount int
	for rows.Next() {
		role := repositories.Role{
			BaseModel: repositories.NewModelBase(),
		}
		err = rows.Scan(append(role.GetScanPointers(), &totalCount)...)
		if err != nil {
			return nil, 0, fmt.Errorf("scanning row: %w", err)
		}

		roles = append(roles, &role)
	}

	return roles, totalCount, nil
}

func (r *roleRepository) Single(ctx context.Context, filter repositories.RoleFilter) (*repositories.Role, error) {
	result, err := r.First(ctx, filter)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, utils.ErrRoleNotFound
	}
	return result, nil
}

func (r *roleRepository) First(ctx context.Context, filter repositories.RoleFilter) (*repositories.Role, error) {
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

	role := repositories.Role{
		BaseModel: repositories.NewModelBase(),
	}
	err = row.Scan(role.GetScanPointers()...)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, nil

	case err != nil:
		return nil, fmt.Errorf("scanning row: %w", err)
	}

	return &role, nil
}

func (r *roleRepository) Insert(ctx context.Context, role *repositories.Role) error {
	scope := middlewares.GetScope(ctx)
	dbService := ioc.GetDependency[database.DbService](scope)

	tx, err := dbService.GetTx()
	if err != nil {
		return fmt.Errorf("failed to open tx: %w", err)
	}

	s := sqlbuilder.InsertInto("roles").
		Cols(
			"virtual_server_id",
			"project_id",
			"name",
			"description",
		).
		Values(
			role.VirtualServerId(),
			role.ProjectId(),
			role.Name(),
			role.Description(),
		).Returning("id", "audit_created_at", "audit_updated_at", "version")

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	row := tx.QueryRowContext(ctx, query, args...)

	err = row.Scan(role.InsertPointers()...)
	if err != nil {
		return fmt.Errorf("scanning row: %w", err)
	}

	role.ClearChanges()
	return nil
}

func (r *roleRepository) Update(ctx context.Context, role *repositories.Role) error {
	scope := middlewares.GetScope(ctx)
	dbService := ioc.GetDependency[database.DbService](scope)

	tx, err := dbService.GetTx()
	if err != nil {
		return fmt.Errorf("failed to open tx: %w", err)
	}

	s := sqlbuilder.Update("roles")
	for fieldName, value := range role.Changes() {
		s.SetMore(s.Assign(fieldName, value))
	}
	s.SetMore(s.Assign("version", role.Version()+1))

	s.Where(s.Equal("id", role.Id()))
	s.Where(s.Equal("version", role.Version()))
	s.Returning("audit_updated_at", "version")

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	row := tx.QueryRowContext(ctx, query, args...)

	err = row.Scan(role.UpdatePointers()...)
	if err != nil {
		return fmt.Errorf("scanning row: %w", err)
	}

	role.ClearChanges()
	return nil
}

func (r *roleRepository) Delete(ctx context.Context, id uuid.UUID) error {
	scope := middlewares.GetScope(ctx)
	dbService := ioc.GetDependency[database.DbService](scope)

	tx, err := dbService.GetTx()
	if err != nil {
		return fmt.Errorf("failed to open tx: %w", err)
	}

	s := sqlbuilder.DeleteFrom("roles")
	s.Where(s.Equal("id", id))

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	_, err = tx.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("executing sql: %w", err)
	}

	return nil
}
