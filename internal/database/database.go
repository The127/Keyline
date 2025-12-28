package database

import (
	"context"
)

type Database interface {
	Migrate(ctx context.Context) error
	NewDbContext(ctx context.Context) (Context, error)
	Close() error
}
