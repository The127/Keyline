package postgres

import (
	"database/sql"
	"github.com/The127/Keyline/internal/change"
)

type GroupRoleRepository struct {
	db            *sql.DB
	changeTracker *change.Tracker
	entityType    int
}

func NewGroupRoleRepository(db *sql.DB, changeTracker *change.Tracker, entityType int) *GroupRoleRepository {
	return &GroupRoleRepository{
		db:            db,
		changeTracker: changeTracker,
		entityType:    entityType,
	}
}
