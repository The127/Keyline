package commands

import (
	"Keyline/internal/middlewares"
	"Keyline/internal/repositories"
	"Keyline/internal/services"
	"Keyline/ioc"
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
	value, err := tokenService.GetToken(ctx, services.EmailVerificationTokenType, command.Token)
	if err != nil {
		return nil, fmt.Errorf("getting token: %w", err)
	}

	userId, err := uuid.Parse(value)
	if err != nil {
		return nil, fmt.Errorf("parsing value: %w", err)
	}

	userRepository := ioc.GetDependency[repositories.UserRepository](scope)
	user, err := userRepository.Single(ctx, repositories.NewUserFilter().Id(userId))
	if err != nil {
		return nil, fmt.Errorf("getting user: %w", err)
	}

	user.SetEmailVerified(true)
	err = userRepository.Update(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("updating user: %w", err)
	}

	err = tokenService.DeleteToken(ctx, services.EmailVerificationTokenType, command.Token)
	if err != nil {
		return nil, fmt.Errorf("deleting token: %w", err)
	}

	return &VerifyEmailResponse{}, nil
}
