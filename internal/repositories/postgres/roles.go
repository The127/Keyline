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

type postgresRole struct {
	postgresBaseModel
	virtualServerId uuid.UUID
	projectId       uuid.UUID
	name            string
	description     string
}

func mapRole(role *repositories.Role) *postgresRole {
	return &postgresRole{
		postgresBaseModel: mapBase(role.BaseModel),
		virtualServerId:   role.VirtualServerId(),
		projectId:         role.ProjectId(),
		name:              role.Name(),
		description:       role.Description(),
	}
}

func (r *postgresRole) Map() *repositories.Role {
	return repositories.NewRoleFromDB(
		r.MapBase(),
		r.virtualServerId,
		r.projectId,
		r.name,
		r.description,
	)
}

func (r *postgresRole) scan(row pghelpers.Row) error {
	return row.Scan(
		&r.id,
		&r.auditCreatedAt,
		&r.auditUpdatedAt,
		&r.xmin,
		&r.virtualServerId,
		&r.projectId,
	)
}

type RoleRepository struct {
	db            *sql.DB
	changeTracker *change.Tracker
	entityType    int
}

func NewRoleRepository(db *sql.DB, changeTracker *change.Tracker, entityType int) *RoleRepository {
	return &RoleRepository{
		db:            db,
		changeTracker: changeTracker,
		entityType:    entityType,
	}
}

func (r *RoleRepository) selectQuery(filter *repositories.RoleFilter) *sqlbuilder.SelectBuilder {
	s := sqlbuilder.Select(
		"id",
		"audit_created_at",
		"audit_updated_at",
		"xmin",
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

func (r *RoleRepository) List(ctx context.Context, filter *repositories.RoleFilter) ([]*repositories.Role, int, error) {
	s := r.selectQuery(filter)
	s.SelectMore("count(*) over()")

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("querying rows: %w", err)
	}
	defer utils.PanicOnError(rows.Close, "closing rows")

	var roles []*repositories.Role
	var totalCount int
	for rows.Next() {
		role := &postgresRole{}
		err := role.scan(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scanning row: %w", err)
		}

		roles = append(roles, role.Map())
	}

	return roles, totalCount, nil
}

func (r *RoleRepository) Single(ctx context.Context, filter *repositories.RoleFilter) (*repositories.Role, error) {
	result, err := r.First(ctx, filter)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, utils.ErrRoleNotFound
	}
	return result, nil
}

func (r *RoleRepository) First(ctx context.Context, filter *repositories.RoleFilter) (*repositories.Role, error) {
	s := r.selectQuery(filter)
	s.Limit(1)

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	row := r.db.QueryRowContext(ctx, query, args...)

	role := &postgresRole{}
	err := role.scan(row)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, nil

	case err != nil:
		return nil, fmt.Errorf("scanning row: %w", err)
	}

	return role.Map(), nil
}

func (r *RoleRepository) Insert(role *repositories.Role) {
	r.changeTracker.Add(change.NewEntry(change.Added, r.entityType, role))
}

func (r *RoleRepository) ExecuteInsert(ctx context.Context, tx *sql.Tx, role *repositories.Role) error {
	mapped := mapRole(role)

	s := sqlbuilder.InsertInto("roles").
		Cols(
			"id",
			"audit_created_at",
			"audit_updated_at",
			"virtual_server_id",
			"project_id",
			"name",
			"description",
		).
		Values(
			mapped.id,
			mapped.auditCreatedAt,
			mapped.auditUpdatedAt,
			mapped.virtualServerId,
			mapped.projectId,
			mapped.name,
			mapped.description,
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

	role.SetVersion(xmin)
	role.ClearChanges()
	return nil
}

func (r *RoleRepository) Update(role *repositories.Role) {
	r.changeTracker.Add(change.NewEntry(change.Updated, r.entityType, role))
}

func (r *RoleRepository) ExecuteUpdate(ctx context.Context, tx *sql.Tx, role *repositories.Role) error {
	if !role.HasChanges() {
		return nil
	}

	mapped := mapRole(role)

	s := sqlbuilder.Update("roles")
	s.Where(s.Equal("id", mapped.id))
	s.Where(s.Equal("xmin", mapped.xmin))

	for _, field := range role.GetChanges() {
		switch field {
		case repositories.RoleChangeName:
			s.SetMore(s.Assign("name", mapped.name))

		case repositories.RoleChangeDescription:
			s.SetMore(s.Assign("description", mapped.description))

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

	role.SetVersion(xmin)
	role.ClearChanges()
	return nil
}

func (r *RoleRepository) Delete(id uuid.UUID) {
	r.changeTracker.Add(change.NewEntry(change.Deleted, r.entityType, id))
}

func (r *RoleRepository) ExecuteDelete(ctx context.Context, tx *sql.Tx, id uuid.UUID) error {
	s := sqlbuilder.DeleteFrom("roles")
	s.Where(s.Equal("id", id))

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	_, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("executing sql: %w", err)
	}

	return nil
}
