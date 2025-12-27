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

type postgresGroup struct {
	postgresBaseModel
	virtualServerId uuid.UUID
	name            string
	description     string
}

func mapGroup(group *repositories.Group) *postgresGroup {
	return &postgresGroup{
		postgresBaseModel: mapBase(group.BaseModel),
		virtualServerId:   group.VirtualServerId(),
		name:              group.Name(),
		description:       group.Description(),
	}
}

func (g *postgresGroup) Map() *repositories.Group {
	return repositories.NewGroupFromDB(
		g.MapBase(),
		g.virtualServerId,
		g.name,
		g.description,
	)
}

func (g *postgresGroup) scan(row pghelpers.Row) error {
	return row.Scan(
		&g.id,
		&g.auditCreatedAt,
		&g.auditUpdatedAt,
		&g.xmin,
		&g.virtualServerId,
		&g.name,
		&g.description,
	)
}

type GroupRepository struct {
	db            *sql.DB
	changeTracker *change.Tracker
	entityType    int
}

func NewGroupRepository(db *sql.DB, changeTracker *change.Tracker, entityType int) *GroupRepository {
	return &GroupRepository{
		db:            db,
		changeTracker: changeTracker,
		entityType:    entityType,
	}
}

func (r *GroupRepository) selectQuery(filter *repositories.GroupFilter) *sqlbuilder.SelectBuilder {
	s := sqlbuilder.Select(
		"id",
		"audit_created_at",
		"audit_updated_at",
		"xmin",
		"virtual_server_id",
		"name",
		"description",
	).From("groups")

	if filter.HasId() {
		s.Where(s.Equal("id", filter.GetId()))
	}

	if filter.HasName() {
		s.Where(s.Equal("name", filter.GetName()))
	}

	if filter.HasVirtualServerId() {
		s.Where(s.Equal("virtual_server_id", filter.GetVirtualServerId()))
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

func (r *GroupRepository) Single(ctx context.Context, filter *repositories.GroupFilter) (*repositories.Group, error) {
	result, err := r.First(ctx, filter)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, utils.ErrGroupNotFound
	}
	return result, nil
}

func (r *GroupRepository) First(ctx context.Context, filter *repositories.GroupFilter) (*repositories.Group, error) {
	s := r.selectQuery(filter)
	s.Limit(1)

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	row := r.db.QueryRowContext(ctx, query, args...)

	group := &postgresGroup{}
	err := group.scan(row)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, nil

	case err != nil:
		return nil, fmt.Errorf("scanning row: %w", err)
	}

	return group.Map(), nil
}

func (r *GroupRepository) List(ctx context.Context, filter *repositories.GroupFilter) ([]*repositories.Group, int, error) {
	s := r.selectQuery(filter)
	s.SelectMore("count(*) over()")

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("querying db: %w", err)
	}
	defer utils.PanicOnError(rows.Close, "closing rows")

	var groups []*repositories.Group
	var totalCount int
	for rows.Next() {
		group := &postgresGroup{}
		err := group.scan(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scanning row: %w", err)
		}

		groups = append(groups, group.Map())
	}

	return groups, totalCount, nil
}

func (r *GroupRepository) Insert(group *repositories.Group) {
	r.changeTracker.Add(change.NewEntry(change.Added, r.entityType, group))
}

func (r *GroupRepository) ExecuteInsert(ctx context.Context, tx *sql.Tx, group *repositories.Group) error {
	mapped := mapGroup(group)

	s := sqlbuilder.InsertInto("groups").
		Cols(
			"id",
			"audit_created_at",
			"audit_updated_at",
			"virtual_server_id",
			"name",
			"description",
		).
		Values(
			mapped.id,
			mapped.auditCreatedAt,
			mapped.auditUpdatedAt,
			mapped.virtualServerId,
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

	group.SetVersion(xmin)
	group.ClearChanges()
	return nil
}

func (r *GroupRepository) Update(group *repositories.Group) {
	r.changeTracker.Add(change.NewEntry(change.Updated, r.entityType, group))
}

func (r *GroupRepository) ExecuteUpdate(ctx context.Context, tx *sql.Tx, group *repositories.Group) error {
	if !group.HasChanges() {
		return nil
	}

	mapped := mapGroup(group)

	s := sqlbuilder.Update("groups")
	s.Where(s.Equal("id", mapped.id))
	s.Where(s.Equal("xmin", mapped.xmin))

	for _, field := range group.GetChanges() {
		switch field {
		case repositories.GroupChangeName:
			s.SetMore(s.Assign("name", mapped.name))

		case repositories.GroupChangeDescription:
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

	group.SetVersion(xmin)
	group.ClearChanges()
	return nil
}

func (r *GroupRepository) Delete(id uuid.UUID) {
	r.changeTracker.Add(change.NewEntry(change.Deleted, r.entityType, id))
}

func (r *GroupRepository) ExecuteDelete(ctx context.Context, tx *sql.Tx, id uuid.UUID) error {
	s := sqlbuilder.DeleteFrom("groups")

	s.Where(s.Equal("id", id))

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	_, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("executing delete: %w", err)
	}

	return nil
}
