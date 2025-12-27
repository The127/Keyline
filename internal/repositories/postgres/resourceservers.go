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

type postgresResourceServer struct {
	postgresBaseModel
	virtualServerId uuid.UUID
	projectId       uuid.UUID
	slug            string
	name            string
	description     string
}

func mapResourceServer(resourceServer *repositories.ResourceServer) *postgresResourceServer {
	return &postgresResourceServer{
		postgresBaseModel: mapBase(resourceServer.BaseModel),
		virtualServerId:   resourceServer.VirtualServerId(),
		projectId:         resourceServer.ProjectId(),
		slug:              resourceServer.Slug(),
		name:              resourceServer.Name(),
		description:       resourceServer.Description(),
	}
}

func (r *postgresResourceServer) Map() *repositories.ResourceServer {
	return repositories.NewResourceServerFromDB(
		r.MapBase(),
		r.virtualServerId,
		r.projectId,
		r.slug,
		r.name,
		r.description,
	)
}

func (r *postgresResourceServer) scan(row pghelpers.Row, additionalPtrs ...any) error {
	ptrs := []any{
		&r.id,
		&r.auditCreatedAt,
		&r.auditUpdatedAt,
		&r.xmin,
		&r.virtualServerId,
		&r.projectId,
		&r.slug,
		&r.name,
		&r.description,
	}

	ptrs = append(ptrs, additionalPtrs...)

	return row.Scan(ptrs...)
}

type ResourceServerRepository struct {
	db            *sql.DB
	changeTracker *change.Tracker
	entityType    int
}

func NewResourceServerRepository(db *sql.DB, changeTracker *change.Tracker, entityType int) *ResourceServerRepository {
	return &ResourceServerRepository{
		db:            db,
		changeTracker: changeTracker,
		entityType:    entityType,
	}
}

func (r *ResourceServerRepository) selectQuery(filter *repositories.ResourceServerFilter) *sqlbuilder.SelectBuilder {
	s := sqlbuilder.Select(
		"id",
		"audit_created_at",
		"audit_updated_at",
		"xmin",
		"virtual_server_id",
		"project_id",
		"slug",
		"name",
		"description",
	).From("resource_servers")

	if filter.HasVirtualServerId() {
		s.Where(s.Equal("virtual_server_id", filter.GetVirtualServerId()))
	}

	if filter.HasProjectId() {
		s.Where(s.Equal("project_id", filter.GetProjectId()))
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

func (r *ResourceServerRepository) List(ctx context.Context, filter *repositories.ResourceServerFilter) ([]*repositories.ResourceServer, int, error) {
	s := r.selectQuery(filter)
	s.SelectMore("count(*) over()")

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("querying db: %w", err)
	}
	defer utils.PanicOnError(rows.Close, "closing rows")

	var resourceServers []*repositories.ResourceServer
	var totalCount int
	for rows.Next() {
		resourceServer := &postgresResourceServer{}
		err := resourceServer.scan(rows, &totalCount)
		if err != nil {
			return nil, 0, fmt.Errorf("scanning row: %w", err)
		}

		resourceServers = append(resourceServers, resourceServer.Map())
	}

	return resourceServers, totalCount, nil
}

func (r *ResourceServerRepository) FirstOrNil(ctx context.Context, filter *repositories.ResourceServerFilter) (*repositories.ResourceServer, error) {
	s := r.selectQuery(filter)
	s.Limit(1)

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	row := r.db.QueryRowContext(ctx, query, args...)

	resourceServer := &postgresResourceServer{}
	err := resourceServer.scan(row)
	if err != nil {
		return nil, fmt.Errorf("scanning row: %w", err)
	}

	return resourceServer.Map(), nil
}

func (r *ResourceServerRepository) FirstOrErr(ctx context.Context, filter *repositories.ResourceServerFilter) (*repositories.ResourceServer, error) {
	resourceServer, err := r.FirstOrNil(ctx, filter)
	if err != nil {
		return nil, err
	}
	if resourceServer == nil {
		return nil, utils.ErrResourceServerNotFound
	}
	return resourceServer, nil
}

func (r *ResourceServerRepository) Insert(resourceServer *repositories.ResourceServer) {
	r.changeTracker.Add(change.NewEntry(change.Added, r.entityType, resourceServer))
}

func (r *ResourceServerRepository) ExecuteInsert(ctx context.Context, tx *sql.Tx, resourceServer *repositories.ResourceServer) error {
	mapped := mapResourceServer(resourceServer)

	s := sqlbuilder.InsertInto("resource_servers").
		Cols(
			"id",
			"audit_created_at",
			"audit_updated_at",
			"virtual_server_id",
			"project_id",
			"slug",
			"name",
			"description",
		).
		Values(
			mapped.id,
			mapped.auditCreatedAt,
			mapped.auditUpdatedAt,
			mapped.virtualServerId,
			mapped.projectId,
			mapped.slug,
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

	resourceServer.SetVersion(xmin)
	resourceServer.ClearChanges()
	return nil
}

func (r *ResourceServerRepository) Update(resourceServer *repositories.ResourceServer) {
	r.changeTracker.Add(change.NewEntry(change.Updated, r.entityType, resourceServer))
}

func (r *ResourceServerRepository) ExecuteUpdate(ctx context.Context, tx *sql.Tx, resourceServer *repositories.ResourceServer) error {
	if !resourceServer.HasChanges() {
		return nil
	}

	mapped := mapResourceServer(resourceServer)

	s := sqlbuilder.Update("resource_servers")
	s.Where(s.Equal("id", mapped.id))
	s.Where(s.Equal("xmin", mapped.xmin))

	for _, field := range resourceServer.GetChanges() {
		switch field {
		case repositories.ResourceServerChangeName:
			s.SetMore(s.Assign("name", mapped.name))

		case repositories.ResourceServerChangeDescription:
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

	resourceServer.SetVersion(xmin)
	resourceServer.ClearChanges()
	return nil
}

func (r *ResourceServerRepository) Delete(id uuid.UUID) {
	r.changeTracker.Add(change.NewEntry(change.Deleted, r.entityType, id))
}

func (r *ResourceServerRepository) ExecuteDelete(ctx context.Context, tx *sql.Tx, id uuid.UUID) error {
	s := sqlbuilder.DeleteFrom("resource_servers")
	s.Where(s.Equal("id", id))

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	_, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("executing sql: %w", err)
	}

	return nil
}
