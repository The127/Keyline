package memory

import (
	"context"
	"github.com/The127/Keyline/internal/change"
	"github.com/The127/Keyline/internal/repositories"
	"github.com/The127/Keyline/utils"
	"sync"

	"github.com/google/uuid"
)

type UserRepository struct {
	store         map[uuid.UUID]*repositories.User
	mu            *sync.RWMutex
	changeTracker *change.Tracker
	entityType    int
}

func NewUserRepository(store map[uuid.UUID]*repositories.User, mu *sync.RWMutex, changeTracker *change.Tracker, entityType int) *UserRepository {
	return &UserRepository{
		store:         store,
		mu:            mu,
		changeTracker: changeTracker,
		entityType:    entityType,
	}
}

func (r *UserRepository) matches(u *repositories.User, filter *repositories.UserFilter) bool {
	if filter.HasId() && u.Id() != filter.GetId() {
		return false
	}
	if filter.HasVirtualServerId() && u.VirtualServerId() != filter.GetVirtualServerId() {
		return false
	}
	if filter.HasUsername() && u.Username() != filter.GetUsername() {
		return false
	}
	if filter.HasServiceUser() && u.IsServiceUser() != filter.GetServiceUser() {
		return false
	}
	if filter.HasSearch() {
		sf := filter.GetSearch()
		if !matchesSearch(u.Username(), sf) && !matchesSearch(u.DisplayName(), sf) {
			return false
		}
	}
	return true
}

func (r *UserRepository) filtered(filter *repositories.UserFilter) []*repositories.User {
	var result []*repositories.User
	for _, u := range r.store {
		if r.matches(u, filter) {
			result = append(result, u)
		}
	}
	return result
}

func (r *UserRepository) FirstOrErr(ctx context.Context, filter *repositories.UserFilter) (*repositories.User, error) {
	result, err := r.FirstOrNil(ctx, filter)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, utils.ErrUserNotFound
	}
	return result, nil
}

func (r *UserRepository) FirstOrNil(_ context.Context, filter *repositories.UserFilter) (*repositories.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	items := r.filtered(filter)
	if len(items) == 0 {
		return nil, nil
	}
	return items[0], nil
}

func (r *UserRepository) List(_ context.Context, filter *repositories.UserFilter) ([]*repositories.User, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	items := r.filtered(filter)
	total := len(items)
	if filter.HasPagination() {
		items = paginateSlice(items, filter.GetPagingInfo())
	}
	return items, total, nil
}

func (r *UserRepository) Insert(user *repositories.User) {
	r.changeTracker.Add(change.NewEntry(change.Added, r.entityType, user))
}

func (r *UserRepository) Update(user *repositories.User) {
	r.changeTracker.Add(change.NewEntry(change.Updated, r.entityType, user))
}
