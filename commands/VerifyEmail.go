package commands

import (
	"Keyline/ioc"
	"Keyline/middlewares"
	"Keyline/repositories"
	"Keyline/services"
	"context"
	"fmt"
	"github.com/google/uuid"
)

type VerifyEmail struct {
	VirtualServerName string
	Token             string
}

type VerifyEmailResponse struct {
}

func HandleVerifyEmail(ctx context.Context, command VerifyEmail) (*VerifyEmailResponse, error) {
	scope := middlewares.GetScope(ctx)

	tokenService := ioc.GetDependency[services.TokenService](scope)
	value, err := tokenService.GetValue(ctx, services.EmailVerificationTokenType, command.Token)
	if err != nil {
		return nil, fmt.Errorf("getting token: %w", err)
	}

	userId, err := uuid.Parse(value)
	if err != nil {
		return nil, fmt.Errorf("parsing value: %w", err)
	}

	userRepository := ioc.GetDependency[*repositories.UserRepository](scope)
	user, err := userRepository.First(ctx, repositories.NewUserFilter().Id(userId))
	if err != nil {
		return nil, fmt.Errorf("getting user: %w", err)
	}

	user.SetEmailVerified(true)
	err = userRepository.Update(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("updating user: %w", err)
	}

	err = tokenService.Delete(ctx, services.EmailVerificationTokenType, command.Token)
	if err != nil {
		return nil, fmt.Errorf("deleting token: %w", err)
	}

	return &VerifyEmailResponse{}, nil
}
