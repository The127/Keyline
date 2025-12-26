package database

import "context"

type Context interface {
	SaveChanges(ctx context.Context) error
}
