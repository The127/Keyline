package commands

import (
	"Keyline/internal/database"
	"Keyline/internal/middlewares"
	"Keyline/internal/repositories"
	"Keyline/internal/services"
	"context"
	"fmt"

	"github.com/The127/ioc"

	"github.com/google/uuid"
)

type VerifyEmail struct {
	VirtualServerName string
	Token             string
}

// Verify email does not implement policy as anyone can verify their email

func (a VerifyEmail) GetRequestName() string {
	return "VerifyEmail"
}

type VerifyEmailResponse struct {
}

func HandleVerifyEmail(ctx context.Context, command VerifyEmail) (*VerifyEmailResponse, error) {
	scope := middlewares.GetScope(ctx)
	dbContext := ioc.GetDependency[database.Context](scope)

	tokenService := ioc.GetDependency[services.TokenService](scope)
	value, err := tokenService.GetToken(ctx, services.EmailVerificationTokenType, command.Token)
	if err != nil {
		return nil, fmt.Errorf("getting token: %w", err)
	}

	userId, err := uuid.Parse(value)
	if err != nil {
		return nil, fmt.Errorf("parsing value: %w", err)
	}

	user, err := dbContext.Users().Single(ctx, repositories.NewUserFilter().Id(userId))
	if err != nil {
		return nil, fmt.Errorf("getting user: %w", err)
	}

	user.SetEmailVerified(true)
	dbContext.Users().Update(user)

	err = tokenService.DeleteToken(ctx, services.EmailVerificationTokenType, command.Token)
	if err != nil {
		return nil, fmt.Errorf("deleting token: %w", err)
	}

	return &VerifyEmailResponse{}, nil
}
