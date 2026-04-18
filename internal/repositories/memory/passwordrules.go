package memory

import (
	"github.com/The127/Keyline/internal/change"
	"github.com/The127/Keyline/internal/repositories"
	"github.com/The127/Keyline/utils"
	"context"
	"sync"

	"github.com/google/uuid"
)

type PasswordRuleRepository struct {
	store         map[uuid.UUID]*repositories.PasswordRule
	mu            *sync.RWMutex
	changeTracker *change.Tracker
	entityType    int
}

func NewPasswordRuleRepository(store map[uuid.UUID]*repositories.PasswordRule, mu *sync.RWMutex, changeTracker *change.Tracker, entityType int) *PasswordRuleRepository {
	return &PasswordRuleRepository{
		store:         store,
		mu:            mu,
		changeTracker: changeTracker,
		entityType:    entityType,
	}
}

func (r *PasswordRuleRepository) matches(p *repositories.PasswordRule, filter *repositories.PasswordRuleFilter) bool {
	if filter.HasVirtualServerId() && p.VirtualServerId() != filter.GetVirtualServerId() {
		return false
	}
	if filter.HasType() && p.Type() != filter.GetType() {
		return false
	}
	return true
}

func (r *PasswordRuleRepository) filtered(filter *repositories.PasswordRuleFilter) []*repositories.PasswordRule {
	var result []*repositories.PasswordRule
	for _, p := range r.store {
		if r.matches(p, filter) {
			result = append(result, p)
		}
	}
	return result
}

func (r *PasswordRuleRepository) FirstOrErr(ctx context.Context, filter *repositories.PasswordRuleFilter) (*repositories.PasswordRule, error) {
	result, err := r.FirstOrNil(ctx, filter)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, utils.ErrPasswordRuleNotFound
	}
	return result, nil
}

func (r *PasswordRuleRepository) FirstOrNil(_ context.Context, filter *repositories.PasswordRuleFilter) (*repositories.PasswordRule, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	items := r.filtered(filter)
	if len(items) == 0 {
		return nil, nil
	}
	return items[0], nil
}

func (r *PasswordRuleRepository) List(_ context.Context, filter *repositories.PasswordRuleFilter) ([]*repositories.PasswordRule, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.filtered(filter), nil
}

func (r *PasswordRuleRepository) Insert(passwordRule *repositories.PasswordRule) {
	r.changeTracker.Add(change.NewEntry(change.Added, r.entityType, passwordRule))
}

func (r *PasswordRuleRepository) Update(passwordRule *repositories.PasswordRule) {
	r.changeTracker.Add(change.NewEntry(change.Updated, r.entityType, passwordRule))
}

func (r *PasswordRuleRepository) Delete(id uuid.UUID) {
	r.changeTracker.Add(change.NewEntry(change.Deleted, r.entityType, id))
}
