package queries

import (
	"github.com/The127/Keyline/internal/database"
	"github.com/The127/Keyline/internal/middlewares"
	"github.com/The127/Keyline/internal/repositories"
	"context"
	"fmt"

	"github.com/The127/ioc"
)

type AnyVirtualServerExists struct{}

// This is only used for initial setup, so we don't care about a policy.

type AnyVirtualServerExistsResult struct {
	Found bool
}

func HandleAnyVirtualServerExists(ctx context.Context, _ AnyVirtualServerExists) (*AnyVirtualServerExistsResult, error) {
	scope := middlewares.GetScope(ctx)
	dbContext := ioc.GetDependency[database.Context](scope)

	virtualServer, err := dbContext.VirtualServers().FirstOrNil(ctx, repositories.NewVirtualServerFilter())
	if err != nil {
		return nil, fmt.Errorf("searching virtual servers: %w", err)
	}

	return &AnyVirtualServerExistsResult{
		Found: virtualServer != nil,
	}, nil
}
