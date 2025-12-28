package commands

import (
	"Keyline/internal/authentication/permissions"
	"Keyline/internal/behaviours"
	db "Keyline/internal/database"
	"Keyline/internal/middlewares"
	"Keyline/internal/repositories"
	"context"
	"fmt"

	"github.com/The127/ioc"

	"github.com/google/uuid"
)

type PatchUser struct {
	VirtualServerName string
	UserId            uuid.UUID
	DisplayName       *string
}

func (a PatchUser) LogRequest() bool {
	return true
}

func (a PatchUser) LogResponse() bool {
	return true
}

func (a PatchUser) IsAllowed(ctx context.Context) (behaviours.PolicyResult, error) {
	return behaviours.PermissionBasedPolicy(ctx, permissions.UserUpdate)
}

func (a PatchUser) GetRequestName() string {
	return "PatchUser"
}

type PatchUserResponse struct{}

func HandlePatchUser(ctx context.Context, command PatchUser) (*PatchUserResponse, error) {
	scope := middlewares.GetScope(ctx)
	dbContext := ioc.GetDependency[db.Context](scope)

	virtualServerFilter := repositories.NewVirtualServerFilter().Name(command.VirtualServerName)
	virtualServer, err := dbContext.VirtualServers().FirstOrErr(ctx, virtualServerFilter)
	if err != nil {
		return nil, fmt.Errorf("getting virtual server: %w", err)
	}

	userFilter := repositories.NewUserFilter().Id(command.UserId).VirtualServerId(virtualServer.Id())
	user, err := dbContext.Users().FirstOrErr(ctx, userFilter)
	if err != nil {
		return nil, fmt.Errorf("getting user: %w", err)
	}

	if command.DisplayName != nil {
		user.SetDisplayName(*command.DisplayName)
	}

	dbContext.Users().Update(user)
	return &PatchUserResponse{}, nil
}
