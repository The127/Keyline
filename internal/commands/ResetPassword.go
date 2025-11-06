package commands

import (
	"Keyline/internal/authentication/permissions"
	"Keyline/internal/behaviours"
	"Keyline/internal/events"
	"Keyline/internal/middlewares"
	"Keyline/internal/repositories"
	"Keyline/ioc"
	"Keyline/utils"
	"context"
	"fmt"
	"github.com/The127/mediatr"

	"github.com/google/uuid"
)

type ResetPassword struct {
	UserId      uuid.UUID
	NewPassword string
	Temporary   bool
}

func (a ResetPassword) LogRequest() bool {
	return true
}

func (a ResetPassword) LogResponse() bool {
	return true
}

func (a ResetPassword) IsAllowed(ctx context.Context) (behaviours.PolicyResult, error) {
	// TODO: users should be able to reset their own password
	return behaviours.PermissionBasedPolicy(ctx, permissions.UserResetPassword)
}

func (a ResetPassword) GetRequestName() string {
	return "ResetPassword"
}

type ResetPasswordResponse struct{}

func HandleResetPassword(ctx context.Context, command ResetPassword) (*ResetPasswordResponse, error) {
	scope := middlewares.GetScope(ctx)

	hashedPassword := utils.HashPassword(command.NewPassword)

	credentialRepository := ioc.GetDependency[repositories.CredentialRepository](scope)
	credentialFilter := repositories.NewCredentialFilter().
		UserId(command.UserId).
		Type(repositories.CredentialTypePassword)
	credential, err := credentialRepository.Single(ctx, credentialFilter)
	if err != nil {
		return nil, fmt.Errorf("getting credential: %w", err)
	}

	details, err := credential.PasswordDetails()
	if err != nil {
		return nil, fmt.Errorf("getting password details: %w", err)
	}
	details.Temporary = command.Temporary
	details.HashedPassword = hashedPassword
	credential.SetDetails(details)

	err = credentialRepository.Update(ctx, credential)
	if err != nil {
		return nil, fmt.Errorf("updating credential: %w", err)
	}

	m := ioc.GetDependency[mediatr.Mediator](scope)
	err = mediatr.SendEvent(ctx, m, events.PasswordChangedEvent{
		UserId: command.UserId,
	})
	if err != nil {
		return nil, fmt.Errorf("raising event: %w", err)
	}

	return &ResetPasswordResponse{}, nil
}
