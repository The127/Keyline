package repositories

import (
	"Keyline/internal/config"
	"Keyline/utils"
	"context"

	"github.com/google/uuid"
)

type VirtualServer struct {
	ModelBase

	name        string
	displayName string

	enableRegistration       bool
	require2fa               bool
	requireEmailVerification bool

	signingAlgorithm config.SigningAlgorithm
}

func NewVirtualServer(name string, displayName string) *VirtualServer {
	return &VirtualServer{
		ModelBase:          NewModelBase(),
		name:               name,
		displayName:        displayName,
		enableRegistration: false,
	}
}

func (m *VirtualServer) GetScanPointers() []any {
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
		&m.signingAlgorithm,
	}
}

func (m *VirtualServer) Name() string {
	return m.name
}

func (m *VirtualServer) DisplayName() string {
	return m.displayName
}

func (m *VirtualServer) SetDisplayName(displayName string) {
	m.displayName = displayName
	m.TrackChange("display_name", displayName)
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

func (m *VirtualServer) SigningAlgorithm() config.SigningAlgorithm {
	return m.signingAlgorithm
}

func (m *VirtualServer) SetSigningAlgorithm(signingAlgorithm config.SigningAlgorithm) {
	m.signingAlgorithm = signingAlgorithm
	m.TrackChange("signing_algorithm", signingAlgorithm)
}

type VirtualServerFilter struct {
	name *string
	id   *uuid.UUID
}

type VirtualServerFilterCacheKey struct {
	name string
	id   uuid.UUID
}

func NewVirtualServerFilter() VirtualServerFilter {
	return VirtualServerFilter{}
}

func (f VirtualServerFilter) GetCacheKey() VirtualServerFilterCacheKey {
	return VirtualServerFilterCacheKey{
		name: utils.ZeroIfNil(f.name),
		id:   utils.ZeroIfNil(f.id),
	}
}

func (f VirtualServerFilter) Clone() VirtualServerFilter {
	return f
}

func (f VirtualServerFilter) Name(name string) VirtualServerFilter {
	filter := f.Clone()
	filter.name = &name
	return filter
}

func (f VirtualServerFilter) HasName() bool {
	return f.name != nil
}

func (f VirtualServerFilter) GetName() string {
	return utils.ZeroIfNil(f.name)
}

func (f VirtualServerFilter) Id(id uuid.UUID) VirtualServerFilter {
	filter := f.Clone()
	filter.id = &id
	return filter
}

func (f VirtualServerFilter) HasId() bool {
	return f.id != nil
}

func (f VirtualServerFilter) GetId() uuid.UUID {
	return utils.ZeroIfNil(f.id)
}

//go:generate mockgen -destination=./mocks/virtualserver_repository.go -package=mocks Keyline/internal/repositories VirtualServerRepository
type VirtualServerRepository interface {
	Single(ctx context.Context, filter VirtualServerFilter) (*VirtualServer, error)
	First(ctx context.Context, filter VirtualServerFilter) (*VirtualServer, error)
	List(ctx context.Context, filter VirtualServerFilter) ([]*VirtualServer, int, error)
	Insert(ctx context.Context, virtualServer *VirtualServer) error
	Update(ctx context.Context, virtualServer *VirtualServer) error
}
