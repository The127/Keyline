package queries

import (
	"context"
	"fmt"
	"github.com/The127/Keyline/internal/authentication/permissions"
	"github.com/The127/Keyline/internal/behaviours"
	"github.com/The127/Keyline/internal/database"
	"github.com/The127/Keyline/internal/middlewares"
	"github.com/The127/Keyline/internal/repositories"
	"time"

	"github.com/The127/ioc"

	"github.com/google/uuid"
)

type GetUserQuery struct {
	UserId            uuid.UUID
	VirtualServerName string
}

func (a GetUserQuery) LogRequest() bool {
	return true
}

func (a GetUserQuery) LogResponse() bool {
	return false
}

func (a GetUserQuery) IsAllowed(ctx context.Context) (behaviours.PolicyResult, error) {
	return behaviours.PermissionBasedPolicy(ctx, permissions.UserView)
}

func (a GetUserQuery) GetRequestName() string {
	return "GetUserQuery"
}

type GetUserQueryResult struct {
	Id            uuid.UUID
	Username      string
	DisplayName   string
	PrimaryEmail  string
	EmailVerified bool
	IsServiceUser bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func HandleGetUserQuery(ctx context.Context, query GetUserQuery) (*GetUserQueryResult, error) {
	scope := middlewares.GetScope(ctx)
	dbContext := ioc.GetDependency[database.Context](scope)

	virtualServerFilter := repositories.NewVirtualServerFilter().Name(query.VirtualServerName)
	virtualServer, err := dbContext.VirtualServers().FirstOrErr(ctx, virtualServerFilter)
	if err != nil {
		return nil, fmt.Errorf("getting virtual server: %w", err)
	}

	userFilter := repositories.NewUserFilter().
		VirtualServerId(virtualServer.Id()).
		Id(query.UserId)
	user, err := dbContext.Users().FirstOrErr(ctx, userFilter)
	if err != nil {
		return nil, fmt.Errorf("getting user: %w", err)
	}

	return &GetUserQueryResult{
		Id:            user.Id(),
		Username:      user.Username(),
		DisplayName:   user.DisplayName(),
		PrimaryEmail:  user.PrimaryEmail(),
		EmailVerified: user.EmailVerified(),
		IsServiceUser: user.IsServiceUser(),
		CreatedAt:     user.AuditCreatedAt(),
		UpdatedAt:     user.AuditUpdatedAt(),
	}, nil
}
