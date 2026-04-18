package memory

import (
	"github.com/The127/Keyline/internal/change"
	"github.com/The127/Keyline/internal/repositories"
	"github.com/The127/Keyline/utils"
	"context"
	"encoding/json"
	"sync"

	"github.com/google/uuid"
)

type CredentialRepository struct {
	store         map[uuid.UUID]*repositories.Credential
	mu            *sync.RWMutex
	changeTracker *change.Tracker
	entityType    int
}

func NewCredentialRepository(store map[uuid.UUID]*repositories.Credential, mu *sync.RWMutex, changeTracker *change.Tracker, entityType int) *CredentialRepository {
	return &CredentialRepository{
		store:         store,
		mu:            mu,
		changeTracker: changeTracker,
		entityType:    entityType,
	}
}

func (r *CredentialRepository) matches(c *repositories.Credential, filter *repositories.CredentialFilter) bool {
	if filter.HasId() && c.Id() != filter.GetId() {
		return false
	}
	if filter.HasUserId() && c.UserId() != filter.GetUserId() {
		return false
	}
	if filter.HasType() && c.Type() != filter.GetType() {
		return false
	}
	if filter.HasDetailKid() || filter.HasDetailPublicKey() || filter.HasDetailsId() {
		// marshal details to JSON to do field-level matching
		detailJson, err := json.Marshal(c.Details())
		if err != nil {
			return false
		}
		var detailMap map[string]any
		if err := json.Unmarshal(detailJson, &detailMap); err != nil {
			return false
		}

		if filter.HasDetailKid() {
			kid, _ := detailMap["kid"].(string)
			if kid != filter.GetDetailKid() {
				return false
			}
		}
		if filter.HasDetailPublicKey() {
			pk, _ := detailMap["publicKey"].(string)
			if pk != filter.GetDetailPublicKey() {
				return false
			}
		}
		if filter.HasDetailsId() {
			id, _ := detailMap["credentialId"].(string)
			if id != filter.GetDetailsId() {
				return false
			}
		}
	}
	return true
}

func (r *CredentialRepository) filtered(filter *repositories.CredentialFilter) []*repositories.Credential {
	var result []*repositories.Credential
	for _, c := range r.store {
		if r.matches(c, filter) {
			result = append(result, c)
		}
	}
	return result
}

func (r *CredentialRepository) FirstOrErr(ctx context.Context, filter *repositories.CredentialFilter) (*repositories.Credential, error) {
	result, err := r.FirstOrNil(ctx, filter)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, utils.ErrCredentialNotFound
	}
	return result, nil
}

func (r *CredentialRepository) FirstOrNil(_ context.Context, filter *repositories.CredentialFilter) (*repositories.Credential, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	items := r.filtered(filter)
	if len(items) == 0 {
		return nil, nil
	}
	return items[0], nil
}

func (r *CredentialRepository) List(_ context.Context, filter *repositories.CredentialFilter) ([]*repositories.Credential, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.filtered(filter), nil
}

func (r *CredentialRepository) Insert(credential *repositories.Credential) {
	r.changeTracker.Add(change.NewEntry(change.Added, r.entityType, credential))
}

func (r *CredentialRepository) Update(credential *repositories.Credential) {
	r.changeTracker.Add(change.NewEntry(change.Updated, r.entityType, credential))
}

func (r *CredentialRepository) Delete(id uuid.UUID) {
	r.changeTracker.Add(change.NewEntry(change.Deleted, r.entityType, id))
}
