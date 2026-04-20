package memory

import (
	"context"
	"github.com/The127/Keyline/internal/change"
	"github.com/The127/Keyline/internal/repositories"
	"sync"

	"github.com/google/uuid"
)

type AuditLogRepository struct {
	store         map[uuid.UUID]*repositories.AuditLog
	mu            *sync.RWMutex
	changeTracker *change.Tracker
	entityType    int
}

func NewAuditLogRepository(store map[uuid.UUID]*repositories.AuditLog, mu *sync.RWMutex, changeTracker *change.Tracker, entityType int) *AuditLogRepository {
	return &AuditLogRepository{
		store:         store,
		mu:            mu,
		changeTracker: changeTracker,
		entityType:    entityType,
	}
}

func (r *AuditLogRepository) matches(a *repositories.AuditLog, filter *repositories.AuditLogFilter) bool {
	if filter.HasVirtualServerId() && a.VirtualServerId() != filter.GetVirtualServerId() {
		return false
	}
	if filter.HasUserId() {
		uid := filter.GetUserId()
		if a.UserId() == nil || *a.UserId() != uid {
			return false
		}
	}
	return true
}

func (r *AuditLogRepository) List(_ context.Context, filter *repositories.AuditLogFilter) ([]*repositories.AuditLog, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var items []*repositories.AuditLog
	for _, a := range r.store {
		if r.matches(a, filter) {
			items = append(items, a)
		}
	}
	total := len(items)
	if filter.HasPagination() {
		items = paginateSlice(items, filter.GetPagingInfo())
	}
	return items, total, nil
}

func (r *AuditLogRepository) Insert(auditLog *repositories.AuditLog) {
	r.changeTracker.Add(change.NewEntry(change.Added, r.entityType, auditLog))
}
