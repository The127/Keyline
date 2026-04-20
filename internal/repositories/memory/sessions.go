package memory

import (
	"context"
	"github.com/The127/Keyline/internal/change"
	"github.com/The127/Keyline/internal/repositories"
	"github.com/The127/Keyline/utils"
	"sync"

	"github.com/google/uuid"
)

type SessionRepository struct {
	store         map[uuid.UUID]*repositories.Session
	mu            *sync.RWMutex
	changeTracker *change.Tracker
	entityType    int
}

func NewSessionRepository(store map[uuid.UUID]*repositories.Session, mu *sync.RWMutex, changeTracker *change.Tracker, entityType int) *SessionRepository {
	return &SessionRepository{
		store:         store,
		mu:            mu,
		changeTracker: changeTracker,
		entityType:    entityType,
	}
}

func (r *SessionRepository) matches(s *repositories.Session, filter *repositories.SessionFilter) bool {
	if filter.HasId() && s.Id() != filter.GetId() {
		return false
	}
	if filter.HasVirtualServerId() && s.VirtualServerId() != filter.GetVirtualServerId() {
		return false
	}
	if filter.HasUserId() && s.UserId() != filter.GetUserId() {
		return false
	}
	return true
}

func (r *SessionRepository) FirstOrErr(ctx context.Context, filter *repositories.SessionFilter) (*repositories.Session, error) {
	result, err := r.FirstOrNil(ctx, filter)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, utils.ErrSessionNotFound
	}
	return result, nil
}

func (r *SessionRepository) FirstOrNil(_ context.Context, filter *repositories.SessionFilter) (*repositories.Session, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, s := range r.store {
		if r.matches(s, filter) {
			return s, nil
		}
	}
	return nil, nil
}

func (r *SessionRepository) Insert(session *repositories.Session) {
	r.changeTracker.Add(change.NewEntry(change.Added, r.entityType, session))
}

func (r *SessionRepository) Delete(id uuid.UUID) {
	r.changeTracker.Add(change.NewEntry(change.Deleted, r.entityType, id))
}
