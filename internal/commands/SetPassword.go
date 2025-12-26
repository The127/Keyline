package commands

import (
	"Keyline/internal/authentication/permissions"
	"Keyline/internal/behaviours"
	"Keyline/internal/database"
	"Keyline/internal/events"
	"Keyline/internal/middlewares"
	"Keyline/internal/repositories"
	"Keyline/utils"
	"context"
	"fmt"

	"github.com/The127/ioc"
	"github.com/The127/mediatr"

	"github.com/google/uuid"
)

type SetPassword struct {
	UserId      uuid.UUID
	NewPassword string
	Temporary   bool
}

func (a SetPassword) LogRequest() bool {
	return true
}

func (a SetPassword) LogResponse() bool {
	return true
}

func (a SetPassword) IsAllowed(ctx context.Context) (behaviours.PolicyResult, error) {
	// TODO: users should be able to reset their own password
	return behaviours.PermissionBasedPolicy(ctx, permissions.UserResetPassword)
}

func (a SetPassword) GetRequestName() string {
	return "SetPassword"
}

type SetPasswordResponse struct{}

func HandleSetPassword(ctx context.Context, command SetPassword) (*SetPasswordResponse, error) {
	scope := middlewares.GetScope(ctx)
	dbContext := ioc.GetDependency[database.Context](scope)

	hashedPassword := utils.HashPassword(command.NewPassword)

	credentialFilter := repositories.NewCredentialFilter().
		UserId(command.UserId).
		Type(repositories.CredentialTypePassword)
	credential, err := dbContext.Credentials().First(ctx, credentialFilter)
	if err != nil {
		return nil, fmt.Errorf("getting credential: %w", err)
	}

	passwordExists := credential != nil

	if !passwordExists {
		credential = repositories.NewCredential(command.UserId, &repositories.CredentialPasswordDetails{})
	}

	details, err := credential.PasswordDetails()
	if err != nil {
		return nil, fmt.Errorf("getting password details: %w", err)
	}

	details.Temporary = command.Temporary
	details.HashedPassword = hashedPassword
	credential.SetDetails(details)

	if passwordExists {
		dbContext.Credentials().Update(credential)
	} else {
		dbContext.Credentials().Insert(credential)
	}

	m := ioc.GetDependency[mediatr.Mediator](scope)
	err = mediatr.SendEvent(ctx, m, events.PasswordChangedEvent{
		UserId: command.UserId,
	})
	if err != nil {
		return nil, fmt.Errorf("raising event: %w", err)
	}

	return &SetPasswordResponse{}, nil
}
