package memory

import (
	"github.com/The127/Keyline/internal/change"
	"github.com/The127/Keyline/internal/repositories"
	"sync"

	"github.com/google/uuid"
)

// GroupRoleRepository implements repositories.GroupRoleRepository.
// The interface has no methods — it is a marker interface.
type GroupRoleRepository struct {
	store         map[uuid.UUID]*repositories.GroupRole
	mu            *sync.RWMutex
	changeTracker *change.Tracker
	entityType    int
}

func NewGroupRoleRepository(store map[uuid.UUID]*repositories.GroupRole, mu *sync.RWMutex, changeTracker *change.Tracker, entityType int) *GroupRoleRepository {
	return &GroupRoleRepository{
		store:         store,
		mu:            mu,
		changeTracker: changeTracker,
		entityType:    entityType,
	}
}
