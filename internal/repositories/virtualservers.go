package repositories

import (
	"Keyline/internal/change"
	"Keyline/internal/config"
	"Keyline/utils"
	"context"

	"github.com/google/uuid"
)

type VirtualServerChange int

const (
	VirtualServerChangeDisplayName VirtualServerChange = iota
	VirtualServerChangeEnableRegistration
	VirtualServerChangeRequire2fa
	VirtualServerChangeRequireEmailVerification
	VirtualServerChangeSigningAlgorithm
)

type VirtualServer struct {
	BaseModel
	change.List[VirtualServerChange]

	name        string
	displayName string

	enableRegistration       bool
	require2fa               bool
	requireEmailVerification bool

	signingAlgorithm config.SigningAlgorithm
}

func NewVirtualServer(name string, displayName string) *VirtualServer {
	return &VirtualServer{
		BaseModel:          NewBaseModel(),
		List:               change.NewChanges[VirtualServerChange](),
		name:               name,
		displayName:        displayName,
		enableRegistration: false,
	}
}

func NewVirtualServerFromDB(base BaseModel, name string, displayName string, enableRegistration bool, require2fa bool, requireEmailVerification bool, signingAlgorithm string) *VirtualServer {
	return &VirtualServer{
		BaseModel:                base,
		List:                     change.NewChanges[VirtualServerChange](),
		name:                     name,
		displayName:              displayName,
		enableRegistration:       enableRegistration,
		require2fa:               require2fa,
		requireEmailVerification: requireEmailVerification,
		signingAlgorithm:         config.SigningAlgorithm(signingAlgorithm),
	}
}

func (m *VirtualServer) Name() string {
	return m.name
}

func (m *VirtualServer) DisplayName() string {
	return m.displayName
}

func (m *VirtualServer) SetDisplayName(displayName string) {
	if m.displayName == displayName {
		return
	}

	m.displayName = displayName
	m.TrackChange(VirtualServerChangeDisplayName)
}

func (m *VirtualServer) EnableRegistration() bool {
	return m.enableRegistration
}

func (m *VirtualServer) SetEnableRegistration(enableRegistration bool) {
	if m.enableRegistration == enableRegistration {
		return
	}

	m.enableRegistration = enableRegistration
	m.TrackChange(VirtualServerChangeEnableRegistration)
}

func (m *VirtualServer) Require2fa() bool {
	return m.require2fa
}

func (m *VirtualServer) SetRequire2fa(require2fa bool) {
	if m.require2fa == require2fa {
		return
	}

	m.require2fa = require2fa
	m.TrackChange(VirtualServerChangeRequire2fa)
}

func (m *VirtualServer) RequireEmailVerification() bool {
	return m.requireEmailVerification
}

func (m *VirtualServer) SetRequireEmailVerification(requireEmailVerification bool) {
	if m.requireEmailVerification == requireEmailVerification {
		return
	}

	m.requireEmailVerification = requireEmailVerification
	m.TrackChange(VirtualServerChangeRequireEmailVerification)
}

func (m *VirtualServer) SigningAlgorithm() config.SigningAlgorithm {
	return m.signingAlgorithm
}

func (m *VirtualServer) SetSigningAlgorithm(signingAlgorithm config.SigningAlgorithm) {
	if m.signingAlgorithm == signingAlgorithm {
		return
	}

	m.signingAlgorithm = signingAlgorithm
	m.TrackChange(VirtualServerChangeSigningAlgorithm)
}

type VirtualServerFilter struct {
	name *string
	id   *uuid.UUID
}

type VirtualServerFilterCacheKey struct {
	name string
	id   uuid.UUID
}

func NewVirtualServerFilter() *VirtualServerFilter {
	return &VirtualServerFilter{}
}

func (f *VirtualServerFilter) GetCacheKey() VirtualServerFilterCacheKey {
	return VirtualServerFilterCacheKey{
		name: utils.ZeroIfNil(f.name),
		id:   utils.ZeroIfNil(f.id),
	}
}

func (f *VirtualServerFilter) Clone() *VirtualServerFilter {
	clone := *f
	return &clone
}

func (f *VirtualServerFilter) Name(name string) *VirtualServerFilter {
	filter := f.Clone()
	filter.name = &name
	return filter
}

func (f *VirtualServerFilter) HasName() bool {
	return f.name != nil
}

func (f *VirtualServerFilter) GetName() string {
	return utils.ZeroIfNil(f.name)
}

func (f *VirtualServerFilter) Id(id uuid.UUID) *VirtualServerFilter {
	filter := f.Clone()
	filter.id = &id
	return filter
}

func (f *VirtualServerFilter) HasId() bool {
	return f.id != nil
}

func (f *VirtualServerFilter) GetId() uuid.UUID {
	return utils.ZeroIfNil(f.id)
}

//go:generate mockgen -destination=./mocks/virtualserver_repository.go -package=mocks Keyline/internal/repositories VirtualServerRepository
type VirtualServerRepository interface {
	Single(ctx context.Context, filter *VirtualServerFilter) (*VirtualServer, error)
	First(ctx context.Context, filter *VirtualServerFilter) (*VirtualServer, error)
	List(ctx context.Context, filter *VirtualServerFilter) ([]*VirtualServer, int, error)
	Insert(virtualServer *VirtualServer)
	Update(virtualServer *VirtualServer)
}
