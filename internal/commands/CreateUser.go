package commands

import (
	"Keyline/internal/authentication/permissions"
	"Keyline/internal/behaviours"
	"Keyline/internal/database"
	"Keyline/internal/events"
	"Keyline/internal/middlewares"
	"Keyline/internal/repositories"
	"context"
	"fmt"

	"github.com/The127/ioc"
	"github.com/The127/mediatr"

	"github.com/google/uuid"
)

type CreateUser struct {
	VirtualServerName string
	Username          string
	DisplayName       string
	Email             string
	EmailVerified     bool
}

func (a CreateUser) LogRequest() bool {
	return true
}

func (a CreateUser) LogResponse() bool {
	return true
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
	dbContext := ioc.GetDependency[database.Context](scope)

	virtualServerFilter := repositories.NewVirtualServerFilter().Name(command.VirtualServerName)
	virtualServer, err := dbContext.VirtualServers().FirstOrErr(ctx, virtualServerFilter)
	if err != nil {
		return nil, fmt.Errorf("getting virtual server: %w", err)
	}

	user := repositories.NewUser(
		command.Username,
		command.DisplayName,
		command.Email,
		virtualServer.Id(),
	)
	user.SetEmailVerified(command.EmailVerified)
	dbContext.Users().Insert(user)

	m := ioc.GetDependency[mediatr.Mediator](scope)
	err = mediatr.SendEvent(ctx, m, events.UserCreatedEvent{
		UserId: user.Id(),
	})
	if err != nil {
		return nil, fmt.Errorf("raising event: %w", err)
	}

	return &CreateUserResponse{
		Id: user.Id(),
	}, nil
}
