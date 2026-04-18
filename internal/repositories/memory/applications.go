package memory

import (
	"github.com/The127/Keyline/internal/change"
	"github.com/The127/Keyline/internal/repositories"
	"github.com/The127/Keyline/utils"
	"context"
	"slices"
	"sync"

	"github.com/google/uuid"
)

type ApplicationRepository struct {
	store         map[uuid.UUID]*repositories.Application
	mu            *sync.RWMutex
	changeTracker *change.Tracker
	entityType    int
}

func NewApplicationRepository(store map[uuid.UUID]*repositories.Application, mu *sync.RWMutex, changeTracker *change.Tracker, entityType int) *ApplicationRepository {
	return &ApplicationRepository{
		store:         store,
		mu:            mu,
		changeTracker: changeTracker,
		entityType:    entityType,
	}
}

func (r *ApplicationRepository) matches(a *repositories.Application, filter *repositories.ApplicationFilter) bool {
	if filter.HasId() && a.Id() != filter.GetId() {
		return false
	}
	if filter.HasIds() {
		if !slices.Contains(filter.GetIds(), a.Id()) {
			return false
		}
	}
	if filter.HasName() && a.Name() != filter.GetName() {
		return false
	}
	if filter.HasVirtualServerId() && a.VirtualServerId() != filter.GetVirtualServerId() {
		return false
	}
	if filter.HasProjectId() && a.ProjectId() != filter.GetProjectId() {
		return false
	}
	if filter.HasSearch() {
		sf := filter.GetSearch()
		if !matchesSearch(a.Name(), sf) && !matchesSearch(a.DisplayName(), sf) {
			return false
		}
	}
	return true
}

func (r *ApplicationRepository) filtered(filter *repositories.ApplicationFilter) []*repositories.Application {
	var result []*repositories.Application
	for _, a := range r.store {
		if r.matches(a, filter) {
			result = append(result, a)
		}
	}
	return result
}

func (r *ApplicationRepository) FirstOrErr(ctx context.Context, filter *repositories.ApplicationFilter) (*repositories.Application, error) {
	result, err := r.FirstOrNil(ctx, filter)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, utils.ErrApplicationNotFound
	}
	return result, nil
}

func (r *ApplicationRepository) FirstOrNil(_ context.Context, filter *repositories.ApplicationFilter) (*repositories.Application, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	items := r.filtered(filter)
	if len(items) == 0 {
		return nil, nil
	}
	return items[0], nil
}

func (r *ApplicationRepository) List(_ context.Context, filter *repositories.ApplicationFilter) ([]*repositories.Application, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	items := r.filtered(filter)
	total := len(items)
	if filter.HasPagination() {
		items = paginateSlice(items, filter.GetPagingInfo())
	}
	return items, total, nil
}

func (r *ApplicationRepository) Insert(application *repositories.Application) {
	r.changeTracker.Add(change.NewEntry(change.Added, r.entityType, application))
}

func (r *ApplicationRepository) Update(application *repositories.Application) {
	r.changeTracker.Add(change.NewEntry(change.Updated, r.entityType, application))
}

func (r *ApplicationRepository) Delete(id uuid.UUID) {
	r.changeTracker.Add(change.NewEntry(change.Deleted, r.entityType, id))
}
