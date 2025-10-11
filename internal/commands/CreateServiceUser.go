package commands

import (
	"Keyline/internal/middlewares"
	"Keyline/internal/repositories"
	"Keyline/ioc"
	"context"
	"fmt"

	"github.com/google/uuid"
)

type CreateServiceUser struct {
	VirtualServerName string
	Username          string
}

type CreateServiceUserResponse struct {
	Id uuid.UUID
}

func HandleCreateServiceUser(ctx context.Context, command CreateServiceUser) (*CreateServiceUserResponse, error) {
	scope := middlewares.GetScope(ctx)

	virtualServerRepository := ioc.GetDependency[repositories.VirtualServerRepository](scope)
	virtualServerFilter := repositories.NewVirtualServerFilter().Name(command.VirtualServerName)
	virtualServer, err := virtualServerRepository.Single(ctx, virtualServerFilter)
	if err != nil {
		return nil, fmt.Errorf("getting virtual server: %w", err)
	}

	userRepository := ioc.GetDependency[repositories.UserRepository](scope)
	user := repositories.NewServiceUser(
		command.Username,
		virtualServer.Id(),
	)
	err = userRepository.Insert(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("inserting user: %w", err)
	}

	return &CreateServiceUserResponse{
		Id: user.Id(),
	}, nil
}
