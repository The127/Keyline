package commands

import (
	"Keyline/internal/authentication/permissions"
	"Keyline/internal/behaviours"
	"Keyline/internal/events"
	"Keyline/internal/middlewares"
	"Keyline/internal/repositories"
	"Keyline/ioc"
	"Keyline/mediator"
	"context"
	"fmt"

	"github.com/google/uuid"
)

type CreateUser struct {
	VirtualServerName string
	Username          string
	DisplayName       string
	Email             string
	EmailVerified     bool
}

func (a CreateUser) IsAllowed(ctx context.Context) (behaviours.PolicyResult, error) {
	return behaviours.PermissionBasedPolicy(ctx, permissions.UserCreate)
}

func (a CreateUser) GetRequestName() string {
	return "CreateUser"
}

type CreateUserResponse struct {
	Id uuid.UUID
}

func HandleCreateUser(ctx context.Context, command CreateUser) (*CreateUserResponse, error) {
	scope := middlewares.GetScope(ctx)

	virtualServerRepository := ioc.GetDependency[repositories.VirtualServerRepository](scope)
	virtualServerFilter := repositories.NewVirtualServerFilter().Name(command.VirtualServerName)
	virtualServer, err := virtualServerRepository.Single(ctx, virtualServerFilter)
	if err != nil {
		return nil, fmt.Errorf("getting virtual server: %w", err)
	}

	userRepository := ioc.GetDependency[repositories.UserRepository](scope)
	user := repositories.NewUser(
		command.Username,
		command.DisplayName,
		command.Email,
		virtualServer.Id(),
	)
	user.SetEmailVerified(command.EmailVerified)
	err = userRepository.Insert(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("inserting user: %w", err)
	}

	m := ioc.GetDependency[mediator.Mediator](scope)
	err = mediator.SendEvent(ctx, m, events.UserCreatedEvent{
		UserId: user.Id(),
	})
	if err != nil {
		return nil, fmt.Errorf("raising event: %w", err)
	}

	return &CreateUserResponse{
		Id: user.Id(),
	}, nil
}
