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

type ApplicationUserMetadataRepository struct {
	store         map[uuid.UUID]*repositories.ApplicationUserMetadata
	mu            *sync.RWMutex
	changeTracker *change.Tracker
	entityType    int
}

func NewApplicationUserMetadataRepository(store map[uuid.UUID]*repositories.ApplicationUserMetadata, mu *sync.RWMutex, changeTracker *change.Tracker, entityType int) *ApplicationUserMetadataRepository {
	return &ApplicationUserMetadataRepository{
		store:         store,
		mu:            mu,
		changeTracker: changeTracker,
		entityType:    entityType,
	}
}

func (r *ApplicationUserMetadataRepository) matches(m *repositories.ApplicationUserMetadata, filter *repositories.ApplicationUserMetadataFilter) bool {
	if filter.HasApplicationId() && m.ApplicationId() != filter.GetApplicationId() {
		return false
	}
	if filter.HasApplicationIds() {
		if !slices.Contains(filter.GetApplicationIds(), m.ApplicationId()) {
			return false
		}
	}
	if filter.HasUserId() && m.UserId() != filter.GetUserId() {
		return false
	}
	return true
}

func (r *ApplicationUserMetadataRepository) filtered(filter *repositories.ApplicationUserMetadataFilter) []*repositories.ApplicationUserMetadata {
	var result []*repositories.ApplicationUserMetadata
	for _, m := range r.store {
		if r.matches(m, filter) {
			result = append(result, m)
		}
	}
	return result
}

func (r *ApplicationUserMetadataRepository) FirstOrErr(_ context.Context, filter *repositories.ApplicationUserMetadataFilter) (*repositories.ApplicationUserMetadata, error) {
	result, err := r.FirstOrNil(nil, filter)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, utils.ErrUserApplicationMetadataNotFound
	}
	return result, nil
}

func (r *ApplicationUserMetadataRepository) FirstOrNil(_ context.Context, filter *repositories.ApplicationUserMetadataFilter) (*repositories.ApplicationUserMetadata, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	items := r.filtered(filter)
	if len(items) == 0 {
		return nil, nil
	}
	return items[0], nil
}

func (r *ApplicationUserMetadataRepository) List(_ context.Context, filter *repositories.ApplicationUserMetadataFilter) ([]*repositories.ApplicationUserMetadata, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	items := r.filtered(filter)
	return items, len(items), nil
}

func (r *ApplicationUserMetadataRepository) Insert(m *repositories.ApplicationUserMetadata) {
	r.changeTracker.Add(change.NewEntry(change.Added, r.entityType, m))
}

func (r *ApplicationUserMetadataRepository) Update(m *repositories.ApplicationUserMetadata) {
	r.changeTracker.Add(change.NewEntry(change.Updated, r.entityType, m))
}
