package postgres

import (
	"Keyline/internal/change"
	"context"
	"database/sql"
)

type Context struct {
	db            *sql.DB
	changeTracker *change.Tracker
}

func newContext(db *sql.DB) *Context {
	return &Context{
		db:            db,
		changeTracker: change.NewTracker(),
	}
}

func (c *Context) SaveChanges(ctx context.Context) error {
	// TODO: implement me
	return nil
}
