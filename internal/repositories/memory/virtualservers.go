package memory

import (
	"github.com/The127/Keyline/internal/change"
	"github.com/The127/Keyline/internal/repositories"
	"github.com/The127/Keyline/utils"
	"context"
	"sync"

	"github.com/google/uuid"
)

type VirtualServerRepository struct {
	store         map[uuid.UUID]*repositories.VirtualServer
	mu            *sync.RWMutex
	changeTracker *change.Tracker
	entityType    int
}

func NewVirtualServerRepository(store map[uuid.UUID]*repositories.VirtualServer, mu *sync.RWMutex, changeTracker *change.Tracker, entityType int) *VirtualServerRepository {
	return &VirtualServerRepository{
		store:         store,
		mu:            mu,
		changeTracker: changeTracker,
		entityType:    entityType,
	}
}

func (r *VirtualServerRepository) matches(vs *repositories.VirtualServer, filter *repositories.VirtualServerFilter) bool {
	if filter.HasName() && vs.Name() != filter.GetName() {
		return false
	}
	if filter.HasId() && vs.Id() != filter.GetId() {
		return false
	}
	return true
}

func (r *VirtualServerRepository) filtered(filter *repositories.VirtualServerFilter) []*repositories.VirtualServer {
	var result []*repositories.VirtualServer
	for _, vs := range r.store {
		if r.matches(vs, filter) {
			result = append(result, vs)
		}
	}
	return result
}

func (r *VirtualServerRepository) FirstOrErr(_ context.Context, filter *repositories.VirtualServerFilter) (*repositories.VirtualServer, error) {
	result, err := r.FirstOrNil(context.TODO(), filter)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, utils.ErrVirtualServerNotFound
	}
	return result, nil
}

func (r *VirtualServerRepository) FirstOrNil(_ context.Context, filter *repositories.VirtualServerFilter) (*repositories.VirtualServer, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	items := r.filtered(filter)
	if len(items) == 0 {
		return nil, nil
	}
	return items[0], nil
}

func (r *VirtualServerRepository) List(_ context.Context, filter *repositories.VirtualServerFilter) ([]*repositories.VirtualServer, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	items := r.filtered(filter)
	return items, len(items), nil
}

func (r *VirtualServerRepository) Insert(virtualServer *repositories.VirtualServer) {
	r.changeTracker.Add(change.NewEntry(change.Added, r.entityType, virtualServer))
}

func (r *VirtualServerRepository) Update(virtualServer *repositories.VirtualServer) {
	r.changeTracker.Add(change.NewEntry(change.Updated, r.entityType, virtualServer))
}
