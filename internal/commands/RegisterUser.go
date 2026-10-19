package commands

import (
	"context"
	"fmt"

	"github.com/The127/Keyline/internal/database"
	"github.com/The127/Keyline/internal/events"
	"github.com/The127/Keyline/internal/middlewares"
	"github.com/The127/Keyline/internal/password"
	"github.com/The127/Keyline/internal/repositories"
	"github.com/The127/Keyline/utils"

	"github.com/The127/mediatr"

	"github.com/google/uuid"
)

type RegisterUser struct {
	VirtualServerName string
	DisplayName       string
	Username          string
	Password          string
	Email             string
}

// Any user (especially someone who is not logged in) must be able to register.
// Because of that we don't need to check permissions here.

type RegisterUserResponse struct {
	Id uuid.UUID
}

func HandleRegisterUser(ctx context.Context, command RegisterUser) (*RegisterUserResponse, error) {
	scope := middlewares.GetScope(ctx)
	dbContext := scope.GetDependency[database.Context]()

	virtualServerFilter := repositories.NewVirtualServerFilter().Name(command.VirtualServerName)
	virtualServer, err := dbContext.VirtualServers().FirstOrErr(ctx, virtualServerFilter)
	if err != nil {
		return nil, fmt.Errorf("getting virtual server: %w", err)
	}

	if !virtualServer.EnableRegistration() {
		return nil, utils.ErrRegistrationNotEnabled
	}

	passwordValidator := scope.GetDependency[password.Validator]()
	err = passwordValidator.Validate(ctx, command.Password)
	if err != nil {
		return nil, fmt.Errorf("password validation: %w", err)
	}

	user := repositories.NewUser(
		command.Username,
		command.DisplayName,
		command.Email,
		virtualServer.Id(),
	)
	dbContext.Users().Insert(user)

	hashedPassword := utils.HashPassword(command.Password)

	credential := repositories.NewCredential(user.Id(), &repositories.CredentialPasswordDetails{
		HashedPassword: hashedPassword,
		Temporary:      false,
	})
	dbContext.Credentials().Insert(credential)

	m := scope.GetDependency[mediatr.Mediator]()
	err = mediatr.SendEvent(ctx, m, events.UserCreatedEvent{
		User: user,
	})
	if err != nil {
		return nil, fmt.Errorf("raising event: %w", err)
	}

	return &RegisterUserResponse{
		Id: user.Id(),
	}, nil
}
