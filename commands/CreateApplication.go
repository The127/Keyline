package commands

import (
	"Keyline/ioc"
	"Keyline/middlewares"
	"Keyline/repositories"
	"context"
	"fmt"
	"github.com/google/uuid"
)

type CreateApplication struct {
	VirtualServerName      string
	Name                   string
	DisplayName            string
	RedirectUris           []string
	PostLogoutRedirectUris []string
}

type CreateApplicationResponse struct {
	Id     uuid.UUID
	Secret string
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
		command.RedirectUris,
	)
	secret := application.GenerateSecret()
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
