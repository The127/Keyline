package memory

import (
	"github.com/The127/Keyline/internal/change"
	"github.com/The127/Keyline/internal/repositories"
	"context"
	"sync"

	"github.com/google/uuid"
)

type OutboxMessageRepository struct {
	store         map[uuid.UUID]*repositories.OutboxMessage
	mu            *sync.RWMutex
	changeTracker *change.Tracker
	entityType    int
}

func NewOutboxMessageRepository(store map[uuid.UUID]*repositories.OutboxMessage, mu *sync.RWMutex, changeTracker *change.Tracker, entityType int) *OutboxMessageRepository {
	return &OutboxMessageRepository{
		store:         store,
		mu:            mu,
		changeTracker: changeTracker,
		entityType:    entityType,
	}
}

func (r *OutboxMessageRepository) List(_ context.Context, filter *repositories.OutboxMessageFilter) ([]*repositories.OutboxMessage, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*repositories.OutboxMessage, 0, len(r.store))
	for _, m := range r.store {
		if filter.HasId() && m.Id() != filter.GetId() {
			continue
		}
		result = append(result, m)
	}
	return result, nil
}

func (r *OutboxMessageRepository) Insert(outboxMessage *repositories.OutboxMessage) {
	r.changeTracker.Add(change.NewEntry(change.Added, r.entityType, outboxMessage))
}

func (r *OutboxMessageRepository) Delete(id uuid.UUID) {
	r.changeTracker.Add(change.NewEntry(change.Deleted, r.entityType, id))
}
