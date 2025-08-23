package repositories

import (
	"Keyline/database"
	"Keyline/ioc"
	"Keyline/logging"
	"Keyline/middlewares"
	"Keyline/utils"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/huandu/go-sqlbuilder"
)

type VirtualServer struct {
	ModelBase

	name        string
	displayName string

	require2fa         bool
	enableRegistration bool
}

func NewVirtualServer(name string, displayName string) *VirtualServer {
	return &VirtualServer{
		ModelBase:          NewModelBase(),
		name:               name,
		displayName:        displayName,
		enableRegistration: false,
	}
}

func (m *VirtualServer) getScanPointers() []any {
	return []any{
		&m.id,
		&m.auditCreatedAt,
		&m.auditUpdatedAt,
		&m.displayName,
		&m.name,
		&m.enableRegistration,
		&m.require2fa,
	}
}

func (m *VirtualServer) Name() string {
	return m.name
}

func (m *VirtualServer) DisplayName() string {
	return m.displayName
}

func (m *VirtualServer) EnableRegistration() bool {
	return m.enableRegistration
}

func (m *VirtualServer) SetEnableRegistration(enableRegistration bool) {
	m.enableRegistration = enableRegistration
	m.TrackChange("enable_registration", enableRegistration)
}

func (m *VirtualServer) Require2fa() bool {
	return m.require2fa
}

func (m *VirtualServer) SetRequire2fa(require2fa bool) {
	m.require2fa = require2fa
	m.TrackChange("require_2fa", require2fa)
}

type VirtualServerFilter struct {
	name *string
	id   *uuid.UUID
}

func NewVirtualServerFilter() VirtualServerFilter {
	return VirtualServerFilter{}
}

func (f VirtualServerFilter) Clone() VirtualServerFilter {
	return f
}

func (f VirtualServerFilter) Name(name string) VirtualServerFilter {
	filter := f.Clone()
	filter.name = &name
	return filter
}

func (f VirtualServerFilter) Id(id uuid.UUID) VirtualServerFilter {
	filter := f.Clone()
	filter.id = &id
	return filter
}

type VirtualServerRepository interface {
	Single(ctx context.Context, filter VirtualServerFilter) (*VirtualServer, error)
	First(ctx context.Context, filter VirtualServerFilter) (*VirtualServer, error)
	Insert(ctx context.Context, virtualServer *VirtualServer) error
}

type virtualServerRepository struct {
}

func NewVirtualServerRepository() VirtualServerRepository {
	return &virtualServerRepository{}
}

func (r *virtualServerRepository) Single(ctx context.Context, filter VirtualServerFilter) (*VirtualServer, error) {
	result, err := r.First(ctx, filter)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, utils.ErrVirtualServerNotFound
	}

	return result, nil
}

func (r *virtualServerRepository) First(ctx context.Context, filter VirtualServerFilter) (*VirtualServer, error) {
	scope := middlewares.GetScope(ctx)
	dbService := ioc.GetDependency[database.DbService](scope)

	tx, err := dbService.GetTx()
	if err != nil {
		return nil, fmt.Errorf("failed to open tx: %w", err)
	}

	s := sqlbuilder.Select(
		"id",
		"audit_created_at",
		"audit_updated_at",
		"display_name",
		"name",
		"enable_registration",
		"require_2fa",
	).From("virtual_servers")

	if filter.name != nil {
		s.Where(s.Equal("name", filter.name))
	}

	s.Limit(1)

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	row := tx.QueryRowContext(ctx, query, args...)

	virtualServer := VirtualServer{
		ModelBase: NewModelBase(),
	}
	err = row.Scan(virtualServer.getScanPointers()...)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, nil

	case err != nil:
		return nil, fmt.Errorf("scanning row: %w", err)
	}

	return &virtualServer, nil
}

func (r *virtualServerRepository) Insert(ctx context.Context, virtualServer *VirtualServer) error {
	scope := middlewares.GetScope(ctx)
	dbService := ioc.GetDependency[database.DbService](scope)

	tx, err := dbService.GetTx()
	if err != nil {
		return fmt.Errorf("failed to open tx: %w", err)
	}

	s := sqlbuilder.InsertInto("virtual_servers").
		Cols("name", "display_name", "enable_registration", "require_2fa").
		Values(
			virtualServer.name,
			virtualServer.displayName,
			virtualServer.enableRegistration,
			virtualServer.require2fa,
		).Returning("id", "audit_created_at", "audit_updated_at")

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	row := tx.QueryRowContext(ctx, query, args...)

	err = row.Scan(&virtualServer.id, &virtualServer.auditCreatedAt, &virtualServer.auditUpdatedAt)
	if err != nil {
		return fmt.Errorf("scanning row: %w", err)
	}

	virtualServer.ClearChanges()
	return nil
}
