package repositories

import (
	"context"
	"github.com/The127/Keyline/internal/change"
	"github.com/The127/Keyline/utils"

	"github.com/google/uuid"
)

type IdentityProviderChange int

type IdentityProvider struct {
	BaseModel
	change.List[IdentityProviderChange]

	virtualServerId uuid.UUID
	name            string
	displayName     string
}

func NewIdentityProvider(virtualServerId uuid.UUID, name string, displayName string) *IdentityProvider {
	return &IdentityProvider{
		BaseModel:       NewBaseModel(),
		List:            change.NewChanges[IdentityProviderChange](),
		virtualServerId: virtualServerId,
		name:            name,
		displayName:     displayName,
	}
}

func NewIdentityProviderFromDB(base BaseModel, virtualServerId uuid.UUID, name string, displayName string) *IdentityProvider {
	return &IdentityProvider{
		BaseModel:       base,
		List:            change.NewChanges[IdentityProviderChange](),
		virtualServerId: virtualServerId,
		name:            name,
		displayName:     displayName,
	}
}

func (p *IdentityProvider) VirtualServerId() uuid.UUID {
	return p.virtualServerId
}

func (p *IdentityProvider) Name() string {
	return p.name
}

func (p *IdentityProvider) DisplayName() string {
	return p.displayName
}

type IdentityProviderFilter struct {
	id              *uuid.UUID
	virtualServerId *uuid.UUID
	name            *string
}

func NewIdentityProviderFilter() *IdentityProviderFilter {
	return &IdentityProviderFilter{}
}

func (f *IdentityProviderFilter) Clone() *IdentityProviderFilter {
	clone := *f
	return &clone
}

func (f *IdentityProviderFilter) Id(id uuid.UUID) *IdentityProviderFilter {
	filter := f.Clone()
	filter.id = &id
	return filter
}

func (f *IdentityProviderFilter) HasId() bool {
	return f.id != nil
}

func (f *IdentityProviderFilter) GetId() uuid.UUID {
	return utils.ZeroIfNil(f.id)
}

func (f *IdentityProviderFilter) VirtualServerId(virtualServerId uuid.UUID) *IdentityProviderFilter {
	filter := f.Clone()
	filter.virtualServerId = &virtualServerId
	return filter
}

func (f *IdentityProviderFilter) HasVirtualServerId() bool {
	return f.virtualServerId != nil
}

func (f *IdentityProviderFilter) GetVirtualServerId() uuid.UUID {
	return utils.ZeroIfNil(f.virtualServerId)
}

func (f *IdentityProviderFilter) Name(name string) *IdentityProviderFilter {
	filter := f.Clone()
	filter.name = &name
	return filter
}

func (f *IdentityProviderFilter) HasName() bool {
	return f.name != nil
}

func (f *IdentityProviderFilter) GetName() string {
	return utils.ZeroIfNil(f.name)
}

//go:generate mockgen -destination=./mocks/identity_provider_repository.go -package=mocks Keyline/internal/repositories IdentityProviderRepository
type IdentityProviderRepository interface {
	FirstOrErr(ctx context.Context, filter *IdentityProviderFilter) (*IdentityProvider, error)
	FirstOrNil(ctx context.Context, filter *IdentityProviderFilter) (*IdentityProvider, error)
	List(ctx context.Context, filter *IdentityProviderFilter) ([]*IdentityProvider, error)
	Insert(identityProvider *IdentityProvider)
}
