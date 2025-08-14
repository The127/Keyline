package commands

import (
	"Keyline/ioc"
	"Keyline/middlewares"
	"Keyline/repositories"
	"Keyline/utils"
	"context"
	"fmt"
	"github.com/google/uuid"
)

type RegisterUser struct {
	VirtualServerName string
	DisplayName       string
	Username          string
}

type RegisterUserResponse struct {
	Id uuid.UUID
}

func HandleRegisterUser(ctx context.Context, command RegisterUser) (*RegisterUserResponse, error) {
	scope := middlewares.GetScope(ctx)

	virtualServerRepository := ioc.GetDependency[*repositories.VirtualServerRepository](scope)
	virtualServerFilter := repositories.NewVirtualServerFilter().Name(command.VirtualServerName)
	virtualServer, err := virtualServerRepository.First(ctx, virtualServerFilter)
	if err != nil {
		return nil, fmt.Errorf("getting virtual server: %w", err)
	}

	if !virtualServer.EnableRegistration() {
		return nil, utils.ErrRegistrationNotEnabled
	}

	userRepository := ioc.GetDependency[*repositories.UserRepository](scope)
	user := repositories.NewUser(
		command.Username,
		command.DisplayName,
		virtualServer.Id(),
	)
	err = userRepository.Insert(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("inserting user: %w", err)
	}

	return &RegisterUserResponse{
		Id: user.Id(),
	}, nil
}
