package repositories

import (
	"Keyline/internal/database"
	"Keyline/internal/logging"
	"Keyline/internal/middlewares"
	"Keyline/ioc"
	"Keyline/utils"
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/huandu/go-sqlbuilder"
)

type Group struct {
	ModelBase

	virtualServerId uuid.UUID

	name        string
	description string
}

func NewGroup(virtualServerId uuid.UUID, name string, description string) *Group {
	return &Group{
		ModelBase:       NewModelBase(),
		virtualServerId: virtualServerId,
		name:            name,
		description:     description,
	}
}

func (g *Group) getScanPointers() []any {
	return []any{
		&g.id,
		&g.auditCreatedAt,
		&g.auditUpdatedAt,
		&g.version,
		&g.virtualServerId,
		&g.name,
		&g.description,
	}
}

func (g *Group) Name() string {
	return g.name
}

func (g *Group) SetName(name string) {
	g.TrackChange("name", name)
	g.name = name
}

func (g *Group) Description() string {
	return g.description
}

func (g *Group) SetDescription(description string) {
	g.TrackChange("description", description)
	g.description = description
}

func (g *Group) VirtualServerId() uuid.UUID {
	return g.virtualServerId
}

type GroupFilter struct {
	name             *string
	virtuallServerId *uuid.UUID
	id               *uuid.UUID
}

func NewGroupFilter() GroupFilter {
	return GroupFilter{}
}

func (f GroupFilter) Clone() GroupFilter {
	return f
}

func (f GroupFilter) Name(name string) GroupFilter {
	filter := f.Clone()
	filter.name = &name
	return filter
}

func (f GroupFilter) VirtualServerId(virtualServerId uuid.UUID) GroupFilter {
	filter := f.Clone()
	filter.virtuallServerId = &virtualServerId
	return filter
}

func (f GroupFilter) Id(id uuid.UUID) GroupFilter {
	filter := f.Clone()
	filter.id = &id
	return filter
}

//go:generate mockgen -destination=./mocks/group_repository.go -package=mocks Keyline/internal/repositories GroupRepository
type GroupRepository interface {
	Single(ctx context.Context, filter GroupFilter) (*Group, error)
	First(ctx context.Context, filter GroupFilter) (*Group, error)
	List(ctx context.Context, filter GroupFilter) ([]*Group, int, error)
	Insert(ctx context.Context, group *Group) error
	Update(ctx context.Context, group *Group) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type groupRepository struct {
}

func NewGroupRepository() GroupRepository {
	return &groupRepository{}
}

func (r *groupRepository) selectQuery(filter GroupFilter) *sqlbuilder.SelectBuilder {
	s := sqlbuilder.Select(
		"id",
		"audit_created_at",
		"audit_updated_at",
		"version",
		"virtual_server_id",
		"name",
		"description",
	).From("groups")

	if filter.id != nil {
		s.Where(s.Equal("id", filter.id))
	}

	if filter.name != nil {
		s.Where(s.Equal("name", filter.name))
	}

	if filter.virtuallServerId != nil {
		s.Where(s.Equal("virtual_server_id", filter.virtuallServerId))
	}

	return s
}

func (r *groupRepository) Update(ctx context.Context, group *Group) error {
	scope := middlewares.GetScope(ctx)
	dbService := ioc.GetDependency[database.DbService](scope)

	tx, err := dbService.GetTx()
	if err != nil {
		return fmt.Errorf("failed to open tx: %w", err)
	}

	s := sqlbuilder.Update("groups")
	for fieldName, value := range group.changes {
		s.SetMore(s.Assign(fieldName, value))
	}
	s.SetMore(s.Assign("version", group.version+1))

	s.Where(s.Equal("id", group.id))
	s.Where(s.Equal("version", group.version))
	s.Returning("audit_updated_at", "version")

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	row := tx.QueryRowContext(ctx, query, args...)

	err = row.Scan(&group.auditUpdatedAt, &group.version)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return fmt.Errorf("updating group: %w", ErrVersionMismatch)
	case err != nil:
		return fmt.Errorf("scanning row: %w", err)
	}

	group.clearChanges()
	return nil
}

func (r *groupRepository) Single(ctx context.Context, filter GroupFilter) (*Group, error) {
	result, err := r.First(ctx, filter)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, utils.ErrGroupNotFound
	}
	return result, nil
}

func (r *groupRepository) First(ctx context.Context, filter GroupFilter) (*Group, error) {
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

	group := Group{
		ModelBase: NewModelBase(),
	}
	err = row.Scan(group.getScanPointers()...)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, nil

	case err != nil:
		return nil, fmt.Errorf("scanning row: %w", err)
	}

	return &group, nil
}

func (r *groupRepository) List(ctx context.Context, filter GroupFilter) ([]*Group, int, error) {
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

	var groups []*Group
	var totalCount int
	for rows.Next() {
		group := Group{
			ModelBase: NewModelBase(),
		}

		err = rows.Scan(append(group.getScanPointers(), &totalCount)...)
		if err != nil {
			return nil, 0, fmt.Errorf("scanning row: %w", err)
		}

		groups = append(groups, &group)
	}

	return groups, totalCount, nil
}

func (r *groupRepository) Insert(ctx context.Context, group *Group) error {
	scope := middlewares.GetScope(ctx)
	dbService := ioc.GetDependency[database.DbService](scope)

	tx, err := dbService.GetTx()
	if err != nil {
		return fmt.Errorf("failed to open tx: %w", err)
	}

	s := sqlbuilder.InsertInto("groups").
		Cols("virtual_server_id", "name", "description").
		Values(
			group.virtualServerId,
			group.name,
			group.description,
		).Returning("id", "audit_created_at", "audit_updated_at", "version")

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	row := tx.QueryRowContext(ctx, query, args...)

	err = row.Scan(&group.id, &group.auditCreatedAt, &group.auditUpdatedAt, &group.version)
	if err != nil {
		return fmt.Errorf("scanning row: %w", err)
	}

	return nil
}

func (r *groupRepository) Delete(ctx context.Context, id uuid.UUID) error {
	scope := middlewares.GetScope(ctx)
	dbService := ioc.GetDependency[database.DbService](scope)

	tx, err := dbService.GetTx()
	if err != nil {
		return fmt.Errorf("failed to open tx: %w", err)
	}

	s := sqlbuilder.DeleteFrom("groups")

	s.Where(s.Equal("id", id))

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	_, err = tx.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("executing delete: %w", err)
	}

	return nil
}
