package memory

import (
	"github.com/The127/Keyline/internal/change"
	"github.com/The127/Keyline/internal/repositories"
	"github.com/The127/Keyline/utils"
	"context"
	"sync"

	"github.com/google/uuid"
)

type ResourceServerScopeRepository struct {
	store         map[uuid.UUID]*repositories.ResourceServerScope
	mu            *sync.RWMutex
	changeTracker *change.Tracker
	entityType    int
}

func NewResourceServerScopeRepository(store map[uuid.UUID]*repositories.ResourceServerScope, mu *sync.RWMutex, changeTracker *change.Tracker, entityType int) *ResourceServerScopeRepository {
	return &ResourceServerScopeRepository{
		store:         store,
		mu:            mu,
		changeTracker: changeTracker,
		entityType:    entityType,
	}
}

func (r *ResourceServerScopeRepository) matches(s *repositories.ResourceServerScope, filter *repositories.ResourceServerScopeFilter) bool {
	if filter.HasId() && s.Id() != filter.GetId() {
		return false
	}
	if filter.HasVirtualServerId() && s.VirtualServerId() != filter.GetVirtualServerId() {
		return false
	}
	if filter.HasProjectId() && s.ProjectId() != filter.GetProjectId() {
		return false
	}
	if filter.HasResourceServerId() && s.ResourceServerId() != filter.GetResourceServerId() {
		return false
	}
	if filter.HasSearch() {
		sf := filter.GetSearch()
		if !matchesSearch(s.Name(), sf) && !matchesSearch(s.Scope(), sf) {
			return false
		}
	}
	return true
}

func (r *ResourceServerScopeRepository) filtered(filter *repositories.ResourceServerScopeFilter) []*repositories.ResourceServerScope {
	var result []*repositories.ResourceServerScope
	for _, s := range r.store {
		if r.matches(s, filter) {
			result = append(result, s)
		}
	}
	return result
}

func (r *ResourceServerScopeRepository) FirstOrErr(ctx context.Context, filter *repositories.ResourceServerScopeFilter) (*repositories.ResourceServerScope, error) {
	result, err := r.FirstOrNil(ctx, filter)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, utils.ErrResourceServerScopeNotFound
	}
	return result, nil
}

func (r *ResourceServerScopeRepository) FirstOrNil(_ context.Context, filter *repositories.ResourceServerScopeFilter) (*repositories.ResourceServerScope, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	items := r.filtered(filter)
	if len(items) == 0 {
		return nil, nil
	}
	return items[0], nil
}

func (r *ResourceServerScopeRepository) List(_ context.Context, filter *repositories.ResourceServerScopeFilter) ([]*repositories.ResourceServerScope, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	items := r.filtered(filter)
	total := len(items)
	if filter.HasPagination() {
		items = paginateSlice(items, filter.GetPagingInfo())
	}
	return items, total, nil
}

func (r *ResourceServerScopeRepository) Insert(scope *repositories.ResourceServerScope) {
	r.changeTracker.Add(change.NewEntry(change.Added, r.entityType, scope))
}

func (r *ResourceServerScopeRepository) Update(scope *repositories.ResourceServerScope) {
	r.changeTracker.Add(change.NewEntry(change.Updated, r.entityType, scope))
}

func (r *ResourceServerScopeRepository) Delete(id uuid.UUID) {
	r.changeTracker.Add(change.NewEntry(change.Deleted, r.entityType, id))
}
