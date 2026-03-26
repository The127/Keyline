package commands

import (
	"Keyline/internal/authentication"
	"Keyline/internal/behaviours"
	"Keyline/internal/database"
	"Keyline/internal/middlewares"
	"Keyline/internal/repositories"
	"Keyline/utils"
	"context"
	"fmt"

	"github.com/The127/ioc"

	"github.com/google/uuid"
)

type ChangeOwnPassword struct {
	VirtualServerName string
	UserId            uuid.UUID
	CurrentPassword   string
	NewPassword       string
}

func (a ChangeOwnPassword) LogRequest() bool {
	return true
}

func (a ChangeOwnPassword) LogResponse() bool {
	return true
}

func (a ChangeOwnPassword) IsAllowed(ctx context.Context) (behaviours.PolicyResult, error) {
	currentUser := authentication.GetCurrentUser(ctx)
	if !currentUser.IsAuthenticated() {
		return behaviours.Denied(currentUser.UserId, uuid.Nil), nil
	}

	if currentUser.UserId != a.UserId {
		return behaviours.Denied(currentUser.UserId, uuid.Nil), nil
	}

	virtualServerName, err := middlewares.GetVirtualServerName(ctx)
	if err != nil {
		return behaviours.PolicyResult{}, fmt.Errorf("getting virtual server name: %w", err)
	}

	scope := middlewares.GetScope(ctx)
	dbContext := ioc.GetDependency[database.Context](scope)

	virtualServerFilter := repositories.NewVirtualServerFilter().Name(virtualServerName)
	virtualServer, err := dbContext.VirtualServers().FirstOrErr(ctx, virtualServerFilter)
	if err != nil {
		return behaviours.PolicyResult{}, fmt.Errorf("getting virtual server: %w", err)
	}

	return behaviours.Allowed(
		currentUser.UserId,
		virtualServer.Id(),
		behaviours.NewAllowedByOwnership(),
	), nil
}

func (a ChangeOwnPassword) GetRequestName() string {
	return "ChangeOwnPassword"
}

type ChangeOwnPasswordResponse struct{}

func HandleChangeOwnPassword(ctx context.Context, command ChangeOwnPassword) (*ChangeOwnPasswordResponse, error) {
	scope := middlewares.GetScope(ctx)
	dbContext := ioc.GetDependency[database.Context](scope)

	// verify current password
	credentialFilter := repositories.NewCredentialFilter().
		UserId(command.UserId).
		Type(repositories.CredentialTypePassword)
	credential, err := dbContext.Credentials().FirstOrNil(ctx, credentialFilter)
	if err != nil {
		return nil, fmt.Errorf("getting credential: %w", err)
	}

	if credential == nil {
		return nil, fmt.Errorf("no password credential found: %w", utils.ErrHttpBadRequest)
	}

	passwordDetails, err := credential.PasswordDetails()
	if err != nil {
		return nil, fmt.Errorf("getting password details: %w", err)
	}

	if !utils.CompareHash(command.CurrentPassword, passwordDetails.HashedPassword) {
		return nil, fmt.Errorf("current password is incorrect: %w", utils.ErrHttpBadRequest)
	}

	// set new password
	hashedPassword := utils.HashPassword(command.NewPassword)
	credential.SetDetails(&repositories.CredentialPasswordDetails{
		HashedPassword: hashedPassword,
		Temporary:      false,
	})

	dbContext.Credentials().Update(credential)

	return &ChangeOwnPasswordResponse{}, nil
}
