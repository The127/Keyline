package memory

import (
	"github.com/The127/Keyline/internal/change"
	"github.com/The127/Keyline/internal/repositories"
	"github.com/The127/Keyline/utils"
	"context"
	"sync"

	"github.com/google/uuid"
)

type ResourceServerRepository struct {
	store         map[uuid.UUID]*repositories.ResourceServer
	mu            *sync.RWMutex
	changeTracker *change.Tracker
	entityType    int
}

func NewResourceServerRepository(store map[uuid.UUID]*repositories.ResourceServer, mu *sync.RWMutex, changeTracker *change.Tracker, entityType int) *ResourceServerRepository {
	return &ResourceServerRepository{
		store:         store,
		mu:            mu,
		changeTracker: changeTracker,
		entityType:    entityType,
	}
}

func (r *ResourceServerRepository) matches(rs *repositories.ResourceServer, filter *repositories.ResourceServerFilter) bool {
	if filter.HasId() && rs.Id() != filter.GetId() {
		return false
	}
	if filter.HasSlug() && rs.Slug() != filter.GetSlug() {
		return false
	}
	if filter.HasVirtualServerId() && rs.VirtualServerId() != filter.GetVirtualServerId() {
		return false
	}
	if filter.HasProjectId() && rs.ProjectId() != filter.GetProjectId() {
		return false
	}
	if filter.HasSearch() {
		sf := filter.GetSearch()
		if !matchesSearch(rs.Name(), sf) && !matchesSearch(rs.Slug(), sf) {
			return false
		}
	}
	return true
}

func (r *ResourceServerRepository) filtered(filter *repositories.ResourceServerFilter) []*repositories.ResourceServer {
	var result []*repositories.ResourceServer
	for _, rs := range r.store {
		if r.matches(rs, filter) {
			result = append(result, rs)
		}
	}
	return result
}

func (r *ResourceServerRepository) FirstOrErr(_ context.Context, filter *repositories.ResourceServerFilter) (*repositories.ResourceServer, error) {
	result, err := r.FirstOrNil(nil, filter)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, utils.ErrResourceServerNotFound
	}
	return result, nil
}

func (r *ResourceServerRepository) FirstOrNil(_ context.Context, filter *repositories.ResourceServerFilter) (*repositories.ResourceServer, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	items := r.filtered(filter)
	if len(items) == 0 {
		return nil, nil
	}
	return items[0], nil
}

func (r *ResourceServerRepository) List(_ context.Context, filter *repositories.ResourceServerFilter) ([]*repositories.ResourceServer, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	items := r.filtered(filter)
	total := len(items)
	if filter.HasPagination() {
		items = paginateSlice(items, filter.GetPagingInfo())
	}
	return items, total, nil
}

func (r *ResourceServerRepository) Insert(resourceServer *repositories.ResourceServer) {
	r.changeTracker.Add(change.NewEntry(change.Added, r.entityType, resourceServer))
}

func (r *ResourceServerRepository) Update(resourceServer *repositories.ResourceServer) {
	r.changeTracker.Add(change.NewEntry(change.Updated, r.entityType, resourceServer))
}

func (r *ResourceServerRepository) Delete(id uuid.UUID) {
	r.changeTracker.Add(change.NewEntry(change.Deleted, r.entityType, id))
}
