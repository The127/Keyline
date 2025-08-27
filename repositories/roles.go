package repositories

import (
	"Keyline/database"
	"Keyline/database/helpers"
	"Keyline/ioc"
	"Keyline/logging"
	"Keyline/middlewares"
	"Keyline/utils"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/huandu/go-sqlbuilder"
)

type Role struct {
	ModelBase

	virtualServerId uuid.UUID
	applicationId   *uuid.UUID

	name        string
	description string

	requireMfa  bool
	maxTokenAge *time.Duration
}

func NewRole(virtualServerId uuid.UUID, applicationId *uuid.UUID, name string, description string) *Role {
	return &Role{
		ModelBase:       NewModelBase(),
		virtualServerId: virtualServerId,
		applicationId:   applicationId,
		name:            name,
		description:     description,
	}
}

func (r *Role) getScanPointers() []any {
	return []any{
		&r.id,
		&r.auditCreatedAt,
		&r.auditUpdatedAt,
		&r.version,
		&r.virtualServerId,
		&r.applicationId,
		&r.name,
		&r.description,
		&r.requireMfa,
		&r.maxTokenAge,
	}
}

func (r *Role) Name() string {
	return r.name
}

func (r *Role) SetName(name string) {
	r.TrackChange("name", name)
	r.name = name
}

func (r *Role) Description() string {
	return r.description
}

func (r *Role) SetDescription(description string) {
	r.TrackChange("description", description)
	r.description = description
}

func (r *Role) VirtualServerId() uuid.UUID {
	return r.virtualServerId
}

func (r *Role) ApplicationId() *uuid.UUID {
	return r.applicationId
}

func (r *Role) RequireMfa() bool {
	return r.requireMfa
}

func (r *Role) SetRequireMfa(requireMfa bool) {
	r.TrackChange("require_mfa", requireMfa)
	r.requireMfa = requireMfa
}

func (r *Role) MaxTokenAge() *time.Duration {
	return r.maxTokenAge
}

func (r *Role) SetMaxTokenAge(maxTokenAge *time.Duration) {
	r.TrackChange("max_token_age", maxTokenAge)
	r.maxTokenAge = maxTokenAge
}

type RoleFilter struct {
	name            *string
	id              *uuid.UUID
	virtualServerId *uuid.UUID
}

func NewRoleFilter() RoleFilter {
	return RoleFilter{}
}

func (f RoleFilter) Clone() RoleFilter {
	return f
}

func (f RoleFilter) Name(name string) RoleFilter {
	filter := f.Clone()
	filter.name = &name
	return filter
}

func (f RoleFilter) Id(id uuid.UUID) RoleFilter {
	filter := f.Clone()
	filter.id = &id
	return filter
}

func (f RoleFilter) VirtualServerId(virtualServerId uuid.UUID) RoleFilter {
	filter := f.Clone()
	filter.virtualServerId = &virtualServerId
	return filter
}

//go:generate mockgen -destination=./mocks/role_repository.go -package=mocks Keyline/repositories RoleRepository
type RoleRepository interface {
	Single(ctx context.Context, filter RoleFilter) (*Role, error)
	First(ctx context.Context, filter RoleFilter) (*Role, error)
	Insert(ctx context.Context, role *Role) error
}

type roleRepository struct {
}

func NewRoleRepository() RoleRepository {
	return &roleRepository{}
}

func (r *roleRepository) selectQuery(filter RoleFilter) *sqlbuilder.SelectBuilder {
	s := sqlbuilder.Select(
		"id",
		"audit_created_at",
		"audit_updated_at",
		"version",
		"virtual_server_id",
		"application_id",
		"name",
		"description",
		"require_mfa",
		"max_token_age",
	).From("roles")

	if filter.name != nil {
		s.Where(s.Equal("name", filter.name))
	}

	if filter.id != nil {
		s.Where(s.Equal("id", filter.id))
	}

	if filter.virtualServerId != nil {
		s.Where(s.Equal("virtual_server_id", filter.virtualServerId))
	}

	return s
}

func (r *roleRepository) Single(ctx context.Context, filter RoleFilter) (*Role, error) {
	result, err := r.First(ctx, filter)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, utils.ErrRoleNotFound
	}
	return result, nil
}

func (r *roleRepository) First(ctx context.Context, filter RoleFilter) (*Role, error) {
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

	role := Role{
		ModelBase: NewModelBase(),
	}
	err = row.Scan(role.getScanPointers()...)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, nil

	case err != nil:
		return nil, fmt.Errorf("scanning row: %w", err)
	}

	return &role, nil
}

func (r *roleRepository) Insert(ctx context.Context, role *Role) error {
	scope := middlewares.GetScope(ctx)
	dbService := ioc.GetDependency[database.DbService](scope)

	tx, err := dbService.GetTx()
	if err != nil {
		return fmt.Errorf("failed to open tx: %w", err)
	}

	s := sqlbuilder.InsertInto("roles").
		Cols(
			"virtual_server_id",
			"application_id",
			"name",
			"description",
			"require_mfa",
			"max_token_age",
		).
		Values(
			role.virtualServerId,
			role.applicationId,
			role.name,
			role.description,
			role.requireMfa,
			helpers.PqIntervalPtr(role.maxTokenAge),
		).Returning("id", "audit_created_at", "audit_updated_at", "version")

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	row := tx.QueryRowContext(ctx, query, args...)

	err = row.Scan(&role.id, &role.auditCreatedAt, &role.auditUpdatedAt, &role.version)
	if err != nil {
		return fmt.Errorf("scanning row: %w", err)
	}

	role.clearChanges()
	return nil
}
