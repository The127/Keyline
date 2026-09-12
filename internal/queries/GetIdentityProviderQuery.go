package queries

import (
	"context"
	"fmt"
	"github.com/The127/Keyline/internal/authentication/permissions"
	"github.com/The127/Keyline/internal/behaviours"
	"github.com/The127/Keyline/internal/database"
	"github.com/The127/Keyline/internal/middlewares"
	"github.com/The127/Keyline/internal/repositories"

	"github.com/The127/ioc"
)

type GetIdentityProvider struct {
	VirtualServerName string
	Name              string
}

func (q GetIdentityProvider) LogRequest() bool {
	return true
}

func (q GetIdentityProvider) LogResponse() bool {
	return false
}

func (q GetIdentityProvider) IsAllowed(ctx context.Context) (behaviours.PolicyResult, error) {
	return behaviours.PermissionBasedPolicy(ctx, permissions.IdentityProviderView)
}

func (q GetIdentityProvider) GetRequestName() string {
	return "GetIdentityProvider"
}

type GetIdentityProviderResult struct {
	Name        string
	DisplayName string
	Settings    repositories.IdentityProviderSettings
}

func HandleGetIdentityProvider(ctx context.Context, query GetIdentityProvider) (*GetIdentityProviderResult, error) {
	scope := middlewares.GetScope(ctx)
	dbContext := ioc.GetDependency[database.Context](scope)

	virtualServerFilter := repositories.NewVirtualServerFilter().Name(query.VirtualServerName)
	virtualServer, err := dbContext.VirtualServers().FirstOrErr(ctx, virtualServerFilter)
	if err != nil {
		return nil, fmt.Errorf("getting virtual server: %w", err)
	}

	identityProviderFilter := repositories.NewIdentityProviderFilter().
		VirtualServerId(virtualServer.Id()).
		Name(query.Name)
	identityProvider, err := dbContext.IdentityProviders().FirstOrErr(ctx, identityProviderFilter)
	if err != nil {
		return nil, fmt.Errorf("getting identity provider: %w", err)
	}

	return &GetIdentityProviderResult{
		Name:        identityProvider.Name(),
		DisplayName: identityProvider.DisplayName(),
		Settings:    identityProvider.Settings(),
	}, nil
}
