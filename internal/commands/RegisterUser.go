package commands

import (
	"Keyline/internal/events"
	"Keyline/internal/middlewares"
	"Keyline/internal/repositories"
	"Keyline/ioc"
	"Keyline/mediator"
	"Keyline/utils"
	"context"
	"fmt"

	"github.com/google/uuid"
)

type RegisterUser struct {
	VirtualServerName string
	DisplayName       string
	Username          string
	Password          string
	Email             string
}

type RegisterUserResponse struct {
	Id uuid.UUID
}

func HandleRegisterUser(ctx context.Context, command RegisterUser) (*RegisterUserResponse, error) {
	scope := middlewares.GetScope(ctx)

	virtualServerRepository := ioc.GetDependency[repositories.VirtualServerRepository](scope)
	virtualServerFilter := repositories.NewVirtualServerFilter().Name(command.VirtualServerName)
	virtualServer, err := virtualServerRepository.Single(ctx, virtualServerFilter)
	if err != nil {
		return nil, fmt.Errorf("getting virtual server: %w", err)
	}

	if !virtualServer.EnableRegistration() {
		return nil, utils.ErrRegistrationNotEnabled
	}

	userRepository := ioc.GetDependency[repositories.UserRepository](scope)
	user := repositories.NewUser(
		command.Username,
		command.DisplayName,
		command.Email,
		virtualServer.Id(),
	)
	err = userRepository.Insert(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("inserting user: %w", err)
	}

	hashedPassword := utils.HashPassword(command.Password)

	credentialRepository := ioc.GetDependency[repositories.CredentialRepository](scope)
	credential := repositories.NewCredential(user.Id(), &repositories.CredentialPasswordDetails{
		HashedPassword: hashedPassword,
		Temporary:      false,
	})
	err = credentialRepository.Insert(ctx, credential)
	if err != nil {
		return nil, fmt.Errorf("inserting credential: %w", err)
	}

	m := ioc.GetDependency[mediator.Mediator](scope)
	err = mediator.SendEvent(ctx, m, events.UserCreatedEvent{
		UserId: user.Id(),
	})
	if err != nil {
		return nil, fmt.Errorf("raising event: %w", err)
	}

	return &RegisterUserResponse{
		Id: user.Id(),
	}, nil
}
