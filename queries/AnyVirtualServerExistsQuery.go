package queries

import (
	"Keyline/ioc"
	"Keyline/middlewares"
	"Keyline/repositories"
	"context"
	"fmt"
)

type AnyVirtualServerExists struct{}
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
