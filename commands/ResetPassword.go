package commands

import (
	"Keyline/events"
	"Keyline/ioc"
	"Keyline/mediator"
	"Keyline/middlewares"
	"Keyline/repositories"
	"Keyline/utils"
	"context"
	"fmt"

	"github.com/google/uuid"
)

type ResetPassword struct {
	UserId      uuid.UUID
	NewPassword string
	Temporary   bool
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

	m := ioc.GetDependency[mediator.Mediator](scope)
	err = mediator.SendEvent(ctx, m, events.PasswordChangedEvent{
		UserId: command.UserId,
	})
	if err != nil {
		return nil, fmt.Errorf("raising event: %w", err)
	}

	return &ResetPasswordResponse{}, nil
}
