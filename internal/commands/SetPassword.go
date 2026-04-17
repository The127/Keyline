package commands

import (
	"github.com/The127/Keyline/internal/authentication/permissions"
	"github.com/The127/Keyline/internal/behaviours"
	"github.com/The127/Keyline/internal/database"
	"github.com/The127/Keyline/internal/middlewares"
	"github.com/The127/Keyline/internal/repositories"
	"github.com/The127/Keyline/utils"
	"context"
	"fmt"

	"github.com/The127/ioc"
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
	credential, err := dbContext.Credentials().FirstOrNil(ctx, credentialFilter)
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

	return &SetPasswordResponse{}, nil
}
