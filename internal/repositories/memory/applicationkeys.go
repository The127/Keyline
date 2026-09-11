package memory

import (
	"context"
	"github.com/The127/Keyline/internal/change"
	"github.com/The127/Keyline/internal/repositories"
	"github.com/The127/Keyline/utils"
	"sort"
	"sync"

	"github.com/google/uuid"
)

type ApplicationKeyRepository struct {
	store         map[uuid.UUID]*repositories.ApplicationKey
	mu            *sync.RWMutex
	changeTracker *change.Tracker
	entityType    int
}

func NewApplicationKeyRepository(store map[uuid.UUID]*repositories.ApplicationKey, mu *sync.RWMutex, changeTracker *change.Tracker, entityType int) *ApplicationKeyRepository {
	return &ApplicationKeyRepository{
		store:         store,
		mu:            mu,
		changeTracker: changeTracker,
		entityType:    entityType,
	}
}

func (r *ApplicationKeyRepository) matches(k *repositories.ApplicationKey, filter *repositories.ApplicationKeyFilter) bool {
	if filter.HasId() && k.Id() != filter.GetId() {
		return false
	}
	if filter.HasApplicationId() && k.ApplicationId() != filter.GetApplicationId() {
		return false
	}
	if filter.HasKid() && k.Kid() != filter.GetKid() {
		return false
	}
	return true
}

func (r *ApplicationKeyRepository) filtered(filter *repositories.ApplicationKeyFilter) []*repositories.ApplicationKey {
	var result []*repositories.ApplicationKey
	for _, k := range r.store {
		if r.matches(k, filter) {
			result = append(result, k)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].AuditCreatedAt().Before(result[j].AuditCreatedAt())
	})
	return result
}

func (r *ApplicationKeyRepository) FirstOrErr(ctx context.Context, filter *repositories.ApplicationKeyFilter) (*repositories.ApplicationKey, error) {
	result, err := r.FirstOrNil(ctx, filter)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, utils.ErrApplicationKeyNotFound
	}
	return result, nil
}

func (r *ApplicationKeyRepository) FirstOrNil(_ context.Context, filter *repositories.ApplicationKeyFilter) (*repositories.ApplicationKey, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	items := r.filtered(filter)
	if len(items) == 0 {
		return nil, nil
	}
	return items[0], nil
}

func (r *ApplicationKeyRepository) List(_ context.Context, filter *repositories.ApplicationKeyFilter) ([]*repositories.ApplicationKey, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.filtered(filter), nil
}

func (r *ApplicationKeyRepository) Insert(applicationKey *repositories.ApplicationKey) {
	r.changeTracker.Add(change.NewEntry(change.Added, r.entityType, applicationKey))
}

func (r *ApplicationKeyRepository) Delete(id uuid.UUID) {
	r.changeTracker.Add(change.NewEntry(change.Deleted, r.entityType, id))
}
