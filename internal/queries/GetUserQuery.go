package queries

import (
	"Keyline/internal/middlewares"
	"Keyline/internal/repositories"
	"Keyline/ioc"
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type GetUserQuery struct {
	UserId            uuid.UUID
	VirtualServerName string
}

type GetUserQueryResult struct {
	Id            uuid.UUID
	Username      string
	DisplayName   string
	PrimaryEmail  string
	EmailVerified bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func HandleGetUserQuery(ctx context.Context, query GetUserQuery) (*GetUserQueryResult, error) {
	scope := middlewares.GetScope(ctx)

	virtualServerRepository := ioc.GetDependency[repositories.VirtualServerRepository](scope)
	virtualServerFilter := repositories.NewVirtualServerFilter().Name(query.VirtualServerName)
	virtualServer, err := virtualServerRepository.Single(ctx, virtualServerFilter)
	if err != nil {
		return nil, fmt.Errorf("getting virtual server: %w", err)
	}

	userRepository := ioc.GetDependency[repositories.UserRepository](scope)
	userFilter := repositories.NewUserFilter().
		VirtualServerId(virtualServer.Id()).
		Id(query.UserId)
	user, err := userRepository.Single(ctx, userFilter)
	if err != nil {
		return nil, fmt.Errorf("getting user: %w", err)
	}

	return &GetUserQueryResult{
		Id:            user.Id(),
		Username:      user.Username(),
		DisplayName:   user.DisplayName(),
		PrimaryEmail:  user.PrimaryEmail(),
		EmailVerified: user.EmailVerified(),
		CreatedAt:     user.AuditCreatedAt(),
		UpdatedAt:     user.AuditUpdatedAt(),
	}, nil
}
