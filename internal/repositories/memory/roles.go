package memory

import (
	"github.com/The127/Keyline/internal/change"
	"github.com/The127/Keyline/internal/repositories"
	"github.com/The127/Keyline/utils"
	"context"
	"sync"

	"github.com/google/uuid"
)

type RoleRepository struct {
	store         map[uuid.UUID]*repositories.Role
	mu            *sync.RWMutex
	changeTracker *change.Tracker
	entityType    int
}

func NewRoleRepository(store map[uuid.UUID]*repositories.Role, mu *sync.RWMutex, changeTracker *change.Tracker, entityType int) *RoleRepository {
	return &RoleRepository{
		store:         store,
		mu:            mu,
		changeTracker: changeTracker,
		entityType:    entityType,
	}
}

func (r *RoleRepository) matches(role *repositories.Role, filter *repositories.RoleFilter) bool {
	if filter.HasId() && role.Id() != filter.GetId() {
		return false
	}
	if filter.HasName() && role.Name() != filter.GetName() {
		return false
	}
	if filter.HasVirtualServerId() && role.VirtualServerId() != filter.GetVirtualServerId() {
		return false
	}
	if filter.HasProjectId() && role.ProjectId() != filter.GetProjectId() {
		return false
	}
	if filter.HasSearch() {
		sf := filter.GetSearch()
		if !matchesSearch(role.Name(), sf) && !matchesSearch(role.Description(), sf) {
			return false
		}
	}
	return true
}

func (r *RoleRepository) filtered(filter *repositories.RoleFilter) []*repositories.Role {
	var result []*repositories.Role
	for _, role := range r.store {
		if r.matches(role, filter) {
			result = append(result, role)
		}
	}
	return result
}

func (r *RoleRepository) FirstOrErr(_ context.Context, filter *repositories.RoleFilter) (*repositories.Role, error) {
	result, err := r.FirstOrNil(context.TODO(), filter)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, utils.ErrRoleNotFound
	}
	return result, nil
}

func (r *RoleRepository) FirstOrNil(_ context.Context, filter *repositories.RoleFilter) (*repositories.Role, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	items := r.filtered(filter)
	if len(items) == 0 {
		return nil, nil
	}
	return items[0], nil
}

func (r *RoleRepository) List(_ context.Context, filter *repositories.RoleFilter) ([]*repositories.Role, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	items := r.filtered(filter)
	total := len(items)
	if filter.HasPagination() {
		items = paginateSlice(items, filter.GetPagingInfo())
	}
	return items, total, nil
}

func (r *RoleRepository) Insert(role *repositories.Role) {
	r.changeTracker.Add(change.NewEntry(change.Added, r.entityType, role))
}

func (r *RoleRepository) Update(role *repositories.Role) {
	r.changeTracker.Add(change.NewEntry(change.Updated, r.entityType, role))
}

func (r *RoleRepository) Delete(id uuid.UUID) {
	r.changeTracker.Add(change.NewEntry(change.Deleted, r.entityType, id))
}
