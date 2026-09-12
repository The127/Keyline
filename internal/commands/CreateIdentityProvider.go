package commands

import (
	"context"
	"fmt"
	"github.com/The127/Keyline/internal/authentication/permissions"
	"github.com/The127/Keyline/internal/behaviours"
	"github.com/The127/Keyline/internal/database"
	"github.com/The127/Keyline/internal/middlewares"
	"github.com/The127/Keyline/internal/repositories"

	"github.com/The127/ioc"

	"github.com/google/uuid"
)

type CreateIdentityProvider struct {
	VirtualServerName string
	Name              string
	DisplayName       string
	Preset            string
	Settings          repositories.IdentityProviderSettings
}

func (c CreateIdentityProvider) LogRequest() bool {
	return true
}

func (c CreateIdentityProvider) LogResponse() bool {
	return true
}

func (c CreateIdentityProvider) IsAllowed(ctx context.Context) (behaviours.PolicyResult, error) {
	return behaviours.PermissionBasedPolicy(ctx, permissions.IdentityProviderCreate)
}

func (c CreateIdentityProvider) GetRequestName() string {
	return "CreateIdentityProvider"
}

type CreateIdentityProviderResponse struct {
	Id uuid.UUID
}

func HandleCreateIdentityProvider(ctx context.Context, command CreateIdentityProvider) (*CreateIdentityProviderResponse, error) {
	scope := middlewares.GetScope(ctx)
	dbContext := ioc.GetDependency[database.Context](scope)

	virtualServerFilter := repositories.NewVirtualServerFilter().Name(command.VirtualServerName)
	virtualServer, err := dbContext.VirtualServers().FirstOrErr(ctx, virtualServerFilter)
	if err != nil {
		return nil, fmt.Errorf("getting virtual server: %w", err)
	}

	identityProvider, err := repositories.NewIdentityProviderFromSettings(virtualServer.Id(), command.Name, command.DisplayName, command.Preset, command.Settings)
	if err != nil {
		return nil, err
	}

	dbContext.IdentityProviders().Insert(identityProvider)

	return &CreateIdentityProviderResponse{
		Id: identityProvider.Id(),
	}, nil
}
