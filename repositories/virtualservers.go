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

	enableRegistration       bool
	require2fa               bool
	requireEmailVerification bool
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
		&m.version,
		&m.displayName,
		&m.name,
		&m.enableRegistration,
		&m.require2fa,
		&m.requireEmailVerification,
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

func (m *VirtualServer) RequireEmailVerification() bool {
	return m.requireEmailVerification
}

func (m *VirtualServer) SetRequireEmailVerification(requireEmailVerification bool) {
	m.requireEmailVerification = requireEmailVerification
	m.TrackChange("require_email_verification", requireEmailVerification)
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

func (f VirtualServerFilter) GetName() *string {
	return f.name
}

func (f VirtualServerFilter) Id(id uuid.UUID) VirtualServerFilter {
	filter := f.Clone()
	filter.id = &id
	return filter
}

func (f VirtualServerFilter) GetId() *uuid.UUID {
	return f.id
}

//go:generate mockgen -destination=./mocks/virtualserver_repository.go -package=mocks Keyline/repositories VirtualServerRepository
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

func (r *virtualServerRepository) selectQuery(filter VirtualServerFilter) *sqlbuilder.SelectBuilder {
	s := sqlbuilder.Select(
		"id",
		"audit_created_at",
		"audit_updated_at",
		"version",
		"display_name",
		"name",
		"enable_registration",
		"require_2fa",
		"require_email_verification",
	).From("virtual_servers")

	if filter.name != nil {
		s.Where(s.Equal("name", filter.name))
	}

	return s
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

	s := r.selectQuery(filter)
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
		Cols(
			"name",
			"display_name",
			"enable_registration",
			"require_2fa",
		).
		Values(
			virtualServer.name,
			virtualServer.displayName,
			virtualServer.enableRegistration,
			virtualServer.require2fa,
		).Returning("id", "audit_created_at", "audit_updated_at", "version")

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	row := tx.QueryRowContext(ctx, query, args...)

	err = row.Scan(&virtualServer.id, &virtualServer.auditCreatedAt, &virtualServer.auditUpdatedAt, &virtualServer.version)
	if err != nil {
		return fmt.Errorf("scanning row: %w", err)
	}

	virtualServer.clearChanges()
	return nil
}
