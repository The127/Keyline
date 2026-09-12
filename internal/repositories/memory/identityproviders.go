package memory

import (
	"context"
	"github.com/The127/Keyline/internal/change"
	"github.com/The127/Keyline/internal/repositories"
	"github.com/The127/Keyline/utils"
	"sort"
	"sync"

	"github.com/google/uuid"
)

type IdentityProviderRepository struct {
	store         map[uuid.UUID]*repositories.IdentityProvider
	mu            *sync.RWMutex
	changeTracker *change.Tracker
	entityType    int
}

func NewIdentityProviderRepository(store map[uuid.UUID]*repositories.IdentityProvider, mu *sync.RWMutex, changeTracker *change.Tracker, entityType int) *IdentityProviderRepository {
	return &IdentityProviderRepository{
		store:         store,
		mu:            mu,
		changeTracker: changeTracker,
		entityType:    entityType,
	}
}

func (r *IdentityProviderRepository) matches(p *repositories.IdentityProvider, filter *repositories.IdentityProviderFilter) bool {
	if filter.HasId() && p.Id() != filter.GetId() {
		return false
	}
	if filter.HasVirtualServerId() && p.VirtualServerId() != filter.GetVirtualServerId() {
		return false
	}
	if filter.HasName() && p.Name() != filter.GetName() {
		return false
	}
	return true
}

func (r *IdentityProviderRepository) filtered(filter *repositories.IdentityProviderFilter) []*repositories.IdentityProvider {
	var result []*repositories.IdentityProvider
	for _, p := range r.store {
		if r.matches(p, filter) {
			result = append(result, p)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].AuditCreatedAt().Before(result[j].AuditCreatedAt())
	})
	return result
}

func (r *IdentityProviderRepository) FirstOrErr(ctx context.Context, filter *repositories.IdentityProviderFilter) (*repositories.IdentityProvider, error) {
	result, err := r.FirstOrNil(ctx, filter)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, utils.ErrIdentityProviderNotFound
	}
	return result, nil
}

func (r *IdentityProviderRepository) FirstOrNil(_ context.Context, filter *repositories.IdentityProviderFilter) (*repositories.IdentityProvider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	items := r.filtered(filter)
	if len(items) == 0 {
		return nil, nil
	}
	return items[0], nil
}

func (r *IdentityProviderRepository) List(_ context.Context, filter *repositories.IdentityProviderFilter) ([]*repositories.IdentityProvider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.filtered(filter), nil
}

func (r *IdentityProviderRepository) Insert(identityProvider *repositories.IdentityProvider) {
	r.changeTracker.Add(change.NewEntry(change.Added, r.entityType, identityProvider))
}
