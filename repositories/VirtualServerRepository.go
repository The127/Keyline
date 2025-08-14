package repositories

import (
	"Keyline/ioc"
	"Keyline/logging"
	"Keyline/middlewares"
	"Keyline/services"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"time"
)

type VirtualServer struct {
	id uuid.UUID

	auditCreatedAt time.Time
	auditUpdatedAt time.Time

	name        string
	displayName string

	enableRegistration bool
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

type VirtualServerFilter struct {
	name *string
}

func NewVirtualServerFilter() VirtualServerFilter {
	return VirtualServerFilter{}
}

func (f VirtualServerFilter) Clone() VirtualServerFilter {
	return f
}

func (f VirtualServerFilter) Name(name string) VirtualServerFilter {
	filter := f.Clone()
	f.name = &name
	return filter
}

type VirtualServerRepository struct {
}

func (r *VirtualServerRepository) First(ctx context.Context, filter VirtualServerFilter) (*VirtualServer, error) {
	scope := middlewares.GetScope(ctx)
	dbService := ioc.GetDependency[*services.DbService](scope)

	tx, err := dbService.GetTx()
	if err != nil {
		return nil, fmt.Errorf("failed to open tx: %w", err)
	}

	s := "select id, audit_created_at, audit_updated_at, display_name, name, enable_registration from virtual_servers "
	params := make([]any, 0)

	if filter.name != nil {
		s += fmt.Sprintf(" where name = $%d", len(params)+1)
		params = append(params, filter.name)
	}

	s += " limit 1"

	logging.Logger.Debug("sql: %s", s)
	row := tx.QueryRow(s, params...)

	var virtualServer VirtualServer
	err = row.Scan(&virtualServer.id, &virtualServer.auditCreatedAt, &virtualServer.auditUpdatedAt, &virtualServer.displayName, &virtualServer.name, &virtualServer.enableRegistration)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, nil

	case err != nil:
		return nil, fmt.Errorf("scanning row: %w", err)
	}

	return &virtualServer, nil
}
