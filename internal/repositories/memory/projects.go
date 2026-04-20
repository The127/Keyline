package memory

import (
	"context"
	"github.com/The127/Keyline/internal/change"
	"github.com/The127/Keyline/internal/repositories"
	"github.com/The127/Keyline/utils"
	"sync"

	"github.com/google/uuid"
)

type ProjectRepository struct {
	store         map[uuid.UUID]*repositories.Project
	mu            *sync.RWMutex
	changeTracker *change.Tracker
	entityType    int
}

func NewProjectRepository(store map[uuid.UUID]*repositories.Project, mu *sync.RWMutex, changeTracker *change.Tracker, entityType int) *ProjectRepository {
	return &ProjectRepository{
		store:         store,
		mu:            mu,
		changeTracker: changeTracker,
		entityType:    entityType,
	}
}

func (r *ProjectRepository) matches(p *repositories.Project, filter *repositories.ProjectFilter) bool {
	if filter.HasId() && p.Id() != filter.GetId() {
		return false
	}
	if filter.HasVirtualServerId() && p.VirtualServerId() != filter.GetVirtualServerId() {
		return false
	}
	if filter.HasSlug() && p.Slug() != filter.GetSlug() {
		return false
	}
	if filter.HasSearch() {
		sf := filter.GetSearch()
		if !matchesSearch(p.Name(), sf) && !matchesSearch(p.Slug(), sf) && !matchesSearch(p.Description(), sf) {
			return false
		}
	}
	return true
}

func (r *ProjectRepository) filtered(filter *repositories.ProjectFilter) []*repositories.Project {
	var result []*repositories.Project
	for _, p := range r.store {
		if r.matches(p, filter) {
			result = append(result, p)
		}
	}
	return result
}

func (r *ProjectRepository) FirstOrErr(ctx context.Context, filter *repositories.ProjectFilter) (*repositories.Project, error) {
	result, err := r.FirstOrNil(ctx, filter)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, utils.ErrProjectNotFound
	}
	return result, nil
}

func (r *ProjectRepository) FirstOrNil(_ context.Context, filter *repositories.ProjectFilter) (*repositories.Project, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	items := r.filtered(filter)
	if len(items) == 0 {
		return nil, nil
	}
	return items[0], nil
}

func (r *ProjectRepository) List(_ context.Context, filter *repositories.ProjectFilter) ([]*repositories.Project, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	items := r.filtered(filter)
	total := len(items)
	if filter.HasPagination() {
		items = paginateSlice(items, filter.GetPagingInfo())
	}
	return items, total, nil
}

func (r *ProjectRepository) Insert(project *repositories.Project) {
	r.changeTracker.Add(change.NewEntry(change.Added, r.entityType, project))
}

func (r *ProjectRepository) Update(project *repositories.Project) {
	r.changeTracker.Add(change.NewEntry(change.Updated, r.entityType, project))
}

func (r *ProjectRepository) Delete(id uuid.UUID) {
	r.changeTracker.Add(change.NewEntry(change.Deleted, r.entityType, id))
}
