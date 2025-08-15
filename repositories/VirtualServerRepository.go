package repositories

import (
	"Keyline/database"
	"Keyline/ioc"
	"Keyline/logging"
	"Keyline/middlewares"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/huandu/go-sqlbuilder"
	"time"
)

type VirtualServer struct {
	ModelBase

	name        string
	displayName string

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

func (m *VirtualServer) Id() uuid.UUID {
	return m.id
}

func (m *VirtualServer) AuditCreatedAt() time.Time {
	return m.auditCreatedAt
}

func (m *VirtualServer) AuditUpdatedAt() time.Time {
	return m.auditUpdatedAt
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

func (m *VirtualServer) SetEnableRegistration(enableRegistration bool) *VirtualServer {
	m.enableRegistration = enableRegistration
	return m
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

type VirtualServerRepository struct {
}

func (r *VirtualServerRepository) First(ctx context.Context, filter VirtualServerFilter) (*VirtualServer, error) {
	scope := middlewares.GetScope(ctx)
	dbService := ioc.GetDependency[*database.DbService](scope)

	tx, err := dbService.GetTx()
	if err != nil {
		return nil, fmt.Errorf("failed to open tx: %w", err)
	}

	s := sqlbuilder.Select("id", "audit_created_at", "audit_updated_at", "display_name", "name", "enable_registration").
		From("virtual_servers")

	if filter.name != nil {
		s.Where(s.Equal("name", filter.name))
	}

	s.Limit(1)

	query, args := s.Build()
	logging.Logger.Debug("sql: %s", query)
	row := tx.QueryRow(query, args...)

	virtualServer := VirtualServer{
		ModelBase: NewModelBase(),
	}
	err = row.Scan(&virtualServer.id, &virtualServer.auditCreatedAt, &virtualServer.auditUpdatedAt, &virtualServer.displayName, &virtualServer.name, &virtualServer.enableRegistration)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, nil

	case err != nil:
		return nil, fmt.Errorf("scanning row: %w", err)
	}

	return &virtualServer, nil
}

func (r *VirtualServerRepository) Insert(ctx context.Context, virtualServer *VirtualServer) error {
	scope := middlewares.GetScope(ctx)
	dbService := ioc.GetDependency[*database.DbService](scope)

	tx, err := dbService.GetTx()
	if err != nil {
		return fmt.Errorf("failed to open tx: %w", err)
	}

	s := sqlbuilder.InsertInto("virtual_servers").
		Cols("name", "display_name", "enable_registration").
		Values(
			virtualServer.name,
			virtualServer.displayName,
			virtualServer.enableRegistration,
		).Returning("id", "audit_created_at", "audit_updated_at")

	query, args := s.Build()
	logging.Logger.Debug("sql: %s", query)
	row := tx.QueryRow(query, args...)

	err = row.Scan(&virtualServer.id, &virtualServer.auditCreatedAt, &virtualServer.auditUpdatedAt)
	if err != nil {
		return fmt.Errorf("scanning row: %w", err)
	}

	return nil
}
