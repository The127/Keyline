package memory

import (
	"github.com/The127/Keyline/internal/change"
	"github.com/The127/Keyline/internal/repositories"
	"github.com/The127/Keyline/utils"
	"context"
	"sync"

	"github.com/google/uuid"
)

type TemplateRepository struct {
	store         map[uuid.UUID]*repositories.Template
	mu            *sync.RWMutex
	changeTracker *change.Tracker
	entityType    int
}

func NewTemplateRepository(store map[uuid.UUID]*repositories.Template, mu *sync.RWMutex, changeTracker *change.Tracker, entityType int) *TemplateRepository {
	return &TemplateRepository{
		store:         store,
		mu:            mu,
		changeTracker: changeTracker,
		entityType:    entityType,
	}
}

func (r *TemplateRepository) matches(t *repositories.Template, filter *repositories.TemplateFilter) bool {
	if filter.HasVirtualServerId() && t.VirtualServerId() != filter.GetVirtualServerId() {
		return false
	}
	if filter.HasTemplateType() && t.TemplateType() != filter.GetTemplateType() {
		return false
	}
	return true
}

func (r *TemplateRepository) filtered(filter *repositories.TemplateFilter) []*repositories.Template {
	var result []*repositories.Template
	for _, t := range r.store {
		if r.matches(t, filter) {
			result = append(result, t)
		}
	}
	return result
}

func (r *TemplateRepository) FirstOrErr(_ context.Context, filter *repositories.TemplateFilter) (*repositories.Template, error) {
	result, err := r.FirstOrNil(context.TODO(), filter)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, utils.ErrTemplateNotFound
	}
	return result, nil
}

func (r *TemplateRepository) FirstOrNil(_ context.Context, filter *repositories.TemplateFilter) (*repositories.Template, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	items := r.filtered(filter)
	if len(items) == 0 {
		return nil, nil
	}
	return items[0], nil
}

func (r *TemplateRepository) List(_ context.Context, filter *repositories.TemplateFilter) ([]*repositories.Template, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	items := r.filtered(filter)
	total := len(items)
	if filter.HasPagination() {
		items = paginateSlice(items, filter.GetPagingInfo())
	}
	return items, total, nil
}

func (r *TemplateRepository) Insert(template *repositories.Template) {
	r.changeTracker.Add(change.NewEntry(change.Added, r.entityType, template))
}
