package commands

import (
	"Keyline/internal/authentication/permissions"
	"Keyline/internal/behaviours"
	"Keyline/internal/middlewares"
	"Keyline/internal/repositories"
	"Keyline/ioc"
	"Keyline/utils"
	"context"
	"fmt"

	"github.com/google/uuid"
)

type CreateApplication struct {
	VirtualServerName      string
	Name                   string
	DisplayName            string
	Type                   repositories.ApplicationType
	RedirectUris           []string
	PostLogoutRedirectUris []string

	HashedSecret *string
}

func (c CreateApplication) IsAllowed(ctx context.Context) (behaviours.PolicyResult, error) {
	return behaviours.PermissionBasedPolicy(ctx, permissions.ApplicationCreate)
}

func (c CreateApplication) GetRequestName() string {
	return "CreateApplication"
}

type CreateApplicationResponse struct {
	Id     uuid.UUID
	Secret *string
}

func HandleCreateApplication(ctx context.Context, command CreateApplication) (*CreateApplicationResponse, error) {
	scope := middlewares.GetScope(ctx)

	virtualServerRepository := ioc.GetDependency[repositories.VirtualServerRepository](scope)
	virtualServerFilter := repositories.NewVirtualServerFilter().Name(command.VirtualServerName)
	virtualServer, err := virtualServerRepository.Single(ctx, virtualServerFilter)
	if err != nil {
		return nil, fmt.Errorf("getting virtual server: %w", err)
	}

	applicationRepository := ioc.GetDependency[repositories.ApplicationRepository](scope)
	application := repositories.NewApplication(
		virtualServer.Id(),
		command.Name,
		command.DisplayName,
		command.Type,
		command.RedirectUris,
	)

	var secret *string = nil
	if command.Type == repositories.ApplicationTypeConfidential {
		if command.HashedSecret != nil {
			application.SetHashedSecret(*command.HashedSecret)
			secret = utils.Ptr("pre hashed secret was used")
		} else {
			secret = utils.Ptr(application.GenerateSecret())
		}
	}

	application.SetPostLogoutRedirectUris(command.PostLogoutRedirectUris)

	err = applicationRepository.Insert(ctx, application)
	if err != nil {
		return nil, fmt.Errorf("inserting application: %w", err)
	}

	return &CreateApplicationResponse{
		Id:     application.Id(),
		Secret: secret,
	}, nil
}
