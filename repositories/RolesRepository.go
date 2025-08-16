package repositories

import (
	"Keyline/database"
	"Keyline/database/helpers"
	"Keyline/ioc"
	"Keyline/logging"
	"Keyline/middlewares"
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/huandu/go-sqlbuilder"
	"time"
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
	name *string
	id   *uuid.UUID
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

type RoleRepository struct {
}

func (r *RoleRepository) Insert(ctx context.Context, role *Role) error {
	scope := middlewares.GetScope(ctx)
	dbService := ioc.GetDependency[*database.DbService](scope)

	tx, err := dbService.GetTx()
	if err != nil {
		return fmt.Errorf("failed to open tx: %w", err)
	}

	s := sqlbuilder.InsertInto("roles").
		Cols("virtual_server_id", "application_id", "name", "description", "require_mfa", "max_token_age").
		Values(
			role.virtualServerId,
			role.applicationId,
			role.name,
			role.description,
			role.requireMfa,
			helpers.PqIntervalPtr(role.maxTokenAge),
		).Returning("id", "audit_created_at", "audit_updated_at")

	query, args := s.Build()
	logging.Logger.Debug("sql: %s", query)
	row := tx.QueryRow(query, args...)

	err = row.Scan(&role.id, &role.auditCreatedAt, &role.auditUpdatedAt)
	if err != nil {
		return fmt.Errorf("scanning row: %w", err)
	}

	return nil
}
