package memory

import (
	"github.com/The127/Keyline/internal/change"
	"github.com/The127/Keyline/internal/repositories"
	"github.com/The127/Keyline/utils"
	"context"
	"sync"

	"github.com/google/uuid"
)

type GroupRepository struct {
	store         map[uuid.UUID]*repositories.Group
	mu            *sync.RWMutex
	changeTracker *change.Tracker
	entityType    int
}

func NewGroupRepository(store map[uuid.UUID]*repositories.Group, mu *sync.RWMutex, changeTracker *change.Tracker, entityType int) *GroupRepository {
	return &GroupRepository{
		store:         store,
		mu:            mu,
		changeTracker: changeTracker,
		entityType:    entityType,
	}
}

func (r *GroupRepository) matches(g *repositories.Group, filter *repositories.GroupFilter) bool {
	if filter.HasId() && g.Id() != filter.GetId() {
		return false
	}
	if filter.HasName() && g.Name() != filter.GetName() {
		return false
	}
	if filter.HasVirtualServerId() && g.VirtualServerId() != filter.GetVirtualServerId() {
		return false
	}
	if filter.HasSearch() {
		sf := filter.GetSearch()
		if !matchesSearch(g.Name(), sf) && !matchesSearch(g.Description(), sf) {
			return false
		}
	}
	return true
}

func (r *GroupRepository) filtered(filter *repositories.GroupFilter) []*repositories.Group {
	var result []*repositories.Group
	for _, g := range r.store {
		if r.matches(g, filter) {
			result = append(result, g)
		}
	}
	return result
}

func (r *GroupRepository) FirstOrErr(_ context.Context, filter *repositories.GroupFilter) (*repositories.Group, error) {
	result, err := r.FirstOrNil(context.TODO(), filter)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, utils.ErrGroupNotFound
	}
	return result, nil
}

func (r *GroupRepository) FirstOrNil(_ context.Context, filter *repositories.GroupFilter) (*repositories.Group, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	items := r.filtered(filter)
	if len(items) == 0 {
		return nil, nil
	}
	return items[0], nil
}

func (r *GroupRepository) List(_ context.Context, filter *repositories.GroupFilter) ([]*repositories.Group, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	items := r.filtered(filter)
	total := len(items)
	if filter.HasPagination() {
		items = paginateSlice(items, filter.GetPagingInfo())
	}
	return items, total, nil
}

func (r *GroupRepository) Insert(group *repositories.Group) {
	r.changeTracker.Add(change.NewEntry(change.Added, r.entityType, group))
}

func (r *GroupRepository) Update(group *repositories.Group) {
	r.changeTracker.Add(change.NewEntry(change.Updated, r.entityType, group))
}

func (r *GroupRepository) Delete(id uuid.UUID) {
	r.changeTracker.Add(change.NewEntry(change.Deleted, r.entityType, id))
}
