package postgres

import (
	"Keyline/internal/change"
	"Keyline/internal/repositories"
	"database/sql"
)

type GroupRoleRepository struct {
	db            *sql.DB
	changeTracker *change.Tracker
	entityType    int
}

func NewGroupRoleRepository(db *sql.DB, changeTracker change.Tracker, entityType int) repositories.GroupRoleRepository {
	return &GroupRoleRepository{
		db:            db,
		changeTracker: &changeTracker,
		entityType:    entityType,
	}
}
