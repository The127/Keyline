package queries

import (
	"Keyline/internal/middlewares"
	"Keyline/internal/repositories"
	"Keyline/ioc"
	"context"
	"fmt"
)

type AnyVirtualServerExists struct{}

// This is only used for initial setup, so we don't care about a policy.

type AnyVirtualServerExistsResult struct {
	Found bool
}

func HandleAnyVirtualServerExists(ctx context.Context, _ AnyVirtualServerExists) (*AnyVirtualServerExistsResult, error) {
	scope := middlewares.GetScope(ctx)

	virtualServerRepository := ioc.GetDependency[repositories.VirtualServerRepository](scope)
	virtualServer, err := virtualServerRepository.First(ctx, repositories.NewVirtualServerFilter())
	if err != nil {
		return nil, fmt.Errorf("searching virtual servers: %w", err)
	}

	return &AnyVirtualServerExistsResult{
		Found: virtualServer != nil,
	}, nil
}
