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

type postgresResourceServerScope struct {
	postgresBaseModel
	virtualServerId  uuid.UUID
	projectId        uuid.UUID
	resourceServerId uuid.UUID
	scope            string
	name             string
	description      string
}

func mapResourceServerScope(resourceServerScope *repositories.ResourceServerScope) *postgresResourceServerScope {
	return &postgresResourceServerScope{
		postgresBaseModel: mapBase(resourceServerScope.BaseModel),
		virtualServerId:   resourceServerScope.VirtualServerId(),
		projectId:         resourceServerScope.ProjectId(),
		resourceServerId:  resourceServerScope.ResourceServerId(),
		scope:             resourceServerScope.Scope(),
		name:              resourceServerScope.Name(),
		description:       resourceServerScope.Description(),
	}
}

func (s *postgresResourceServerScope) Map() *repositories.ResourceServerScope {
	return repositories.NewResourceServerScopeFromDB(
		s.MapBase(),
		s.virtualServerId,
		s.projectId,
		s.resourceServerId,
		s.scope,
		s.name,
		s.description,
	)
}

func (s *postgresResourceServerScope) scan(row pghelpers.Row) error {
	return row.Scan(
		&s.id,
		&s.auditCreatedAt,
		&s.auditUpdatedAt,
		&s.xmin,
		&s.virtualServerId,
		&s.projectId,
		&s.resourceServerId,
		&s.scope,
		&s.name,
	)
}

type ResourceServerScopeRepository struct {
	db            *sql.DB
	changeTracker *change.Tracker
	entityType    int
}

func NewResourceServerScopeRepository(db *sql.DB, changeTracker change.Tracker, entityType int) repositories.ResourceServerScopeRepository {
	return &ResourceServerScopeRepository{
		db:            db,
		changeTracker: &changeTracker,
		entityType:    entityType,
	}
}

func (r *ResourceServerScopeRepository) selectQuery(filter repositories.ResourceServerScopeFilter) *sqlbuilder.SelectBuilder {
	s := sqlbuilder.Select(
		"id",
		"audit_created_at",
		"audit_updated_at",
		"xmin",
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

func (r *ResourceServerScopeRepository) List(ctx context.Context, filter repositories.ResourceServerScopeFilter) ([]*repositories.ResourceServerScope, int, error) {
	s := r.selectQuery(filter)
	s.SelectMore("count(*) over()")

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("querying db: %w", err)
	}
	defer utils.PanicOnError(rows.Close, "closing rows")

	var resourceServerScopes []*repositories.ResourceServerScope
	var totalCount int
	for rows.Next() {
		resourceServerScope := &postgresResourceServerScope{}
		err := resourceServerScope.scan(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scanning row: %w", err)
		}
		resourceServerScopes = append(resourceServerScopes, resourceServerScope.Map())
	}

	return resourceServerScopes, totalCount, nil
}

func (r *ResourceServerScopeRepository) First(ctx context.Context, filter repositories.ResourceServerScopeFilter) (*repositories.ResourceServerScope, error) {
	s := r.selectQuery(filter)
	s.Limit(1)

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	row := r.db.QueryRowContext(ctx, query, args...)

	resourceServerScope := &postgresResourceServerScope{}
	err := resourceServerScope.scan(row)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, nil

	case err != nil:
		return nil, fmt.Errorf("scanning row: %w", err)
	}

	return resourceServerScope.Map(), nil
}

func (r *ResourceServerScopeRepository) Single(ctx context.Context, filter repositories.ResourceServerScopeFilter) (*repositories.ResourceServerScope, error) {
	resourceServerScope, err := r.First(ctx, filter)
	if err != nil {
		return nil, err
	}
	if resourceServerScope == nil {
		return nil, utils.ErrResourceServerScopeNotFound
	}
	return resourceServerScope, nil
}

func (r *ResourceServerScopeRepository) Insert(resourceServerScope *repositories.ResourceServerScope) {
	r.changeTracker.Add(change.NewEntry(change.Added, r.entityType, resourceServerScope))
}

func (r *ResourceServerScopeRepository) ExecuteInsert(ctx context.Context, tx *sql.Tx, resourceServerScope *repositories.ResourceServerScope) error {
	mapped := mapResourceServerScope(resourceServerScope)

	s := sqlbuilder.InsertInto("resource_server_scopes").
		Cols(
			"id",
			"audit_created_at",
			"audit_updated_at",
			"virtual_server_id",
			"project_id",
			"resource_server_id",
			"scope",
			"name",
			"description",
		).
		Values(
			mapped.id,
			mapped.auditCreatedAt,
			mapped.auditUpdatedAt,
			mapped.virtualServerId,
			mapped.projectId,
			mapped.resourceServerId,
			mapped.scope,
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

	resourceServerScope.SetVersion(xmin)
	resourceServerScope.ClearChanges()
	return nil
}

func (r *ResourceServerScopeRepository) Update(resourceServerScope *repositories.ResourceServerScope) {
	r.changeTracker.Add(change.NewEntry(change.Updated, r.entityType, resourceServerScope))
}

func (r *ResourceServerScopeRepository) ExecuteUpdate(ctx context.Context, tx *sql.Tx, resourceServerScope *repositories.ResourceServerScope) error {
	if !resourceServerScope.HasChanges() {
		return nil
	}

	mapped := mapResourceServerScope(resourceServerScope)

	s := sqlbuilder.Update("resource_server_scopes")
	s.Where(s.Equal("id", mapped.id))
	s.Where(s.Equal("xmin", mapped.xmin))

	for _, field := range resourceServerScope.GetChanges() {
		switch field {
		case repositories.ResourceServerScopeChangeName:
			s.SetMore(s.Assign("name", mapped.name))

		case repositories.ResourceServerScopeChangeDescription:
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

	resourceServerScope.SetVersion(xmin)
	resourceServerScope.ClearChanges()
	return nil
}

func (r *ResourceServerScopeRepository) Delete(id uuid.UUID) {
	r.changeTracker.Add(change.NewEntry(change.Deleted, r.entityType, id))
}

func (r *ResourceServerScopeRepository) ExecuteDelete(ctx context.Context, tx *sql.Tx, id uuid.UUID) error {
	s := sqlbuilder.DeleteFrom("resource_server_scopes")
	s.Where(s.Equal("id", id))

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	_, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("executing sql: %w", err)
	}

	return nil
}
