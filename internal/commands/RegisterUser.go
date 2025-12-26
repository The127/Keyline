package commands

import (
	"Keyline/internal/database"
	"Keyline/internal/events"
	"Keyline/internal/middlewares"
	"Keyline/internal/password"
	"Keyline/internal/repositories"
	"Keyline/utils"
	"context"
	"fmt"

	"github.com/The127/ioc"
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
	dbContext := ioc.GetDependency[database.Context](scope)

	virtualServerFilter := repositories.NewVirtualServerFilter().Name(command.VirtualServerName)
	virtualServer, err := dbContext.VirtualServers().Single(ctx, virtualServerFilter)
	if err != nil {
		return nil, fmt.Errorf("getting virtual server: %w", err)
	}

	if !virtualServer.EnableRegistration() {
		return nil, utils.ErrRegistrationNotEnabled
	}

	passwordValidator := ioc.GetDependency[password.Validator](scope)
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

	credentialRepository := ioc.GetDependency[repositories.CredentialRepository](scope)
	credential := repositories.NewCredential(user.Id(), &repositories.CredentialPasswordDetails{
		HashedPassword: hashedPassword,
		Temporary:      false,
	})
	credentialRepository.Insert(credential)

	m := ioc.GetDependency[mediatr.Mediator](scope)
	err = mediatr.SendEvent(ctx, m, events.UserCreatedEvent{
		UserId: user.Id(),
	})
	if err != nil {
		return nil, fmt.Errorf("raising event: %w", err)
	}

	return &RegisterUserResponse{
		Id: user.Id(),
	}, nil
}
