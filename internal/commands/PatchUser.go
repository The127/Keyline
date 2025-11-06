package commands

import (
	"Keyline/internal/authentication/permissions"
	"Keyline/internal/behaviours"
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

	virtualServerRepository := ioc.GetDependency[repositories.VirtualServerRepository](scope)
	virtualServerFilter := repositories.NewVirtualServerFilter().Name(command.VirtualServerName)
	virtualServer, err := virtualServerRepository.Single(ctx, virtualServerFilter)
	if err != nil {
		return nil, fmt.Errorf("getting virtual server: %w", err)
	}

	userRepository := ioc.GetDependency[repositories.UserRepository](scope)
	userFilter := repositories.NewUserFilter().Id(command.UserId).VirtualServerId(virtualServer.Id())
	user, err := userRepository.Single(ctx, userFilter)
	if err != nil {
		return nil, fmt.Errorf("getting user: %w", err)
	}

	if command.DisplayName != nil {
		user.SetDisplayName(*command.DisplayName)
	}

	err = userRepository.Update(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("updating user: %w", err)
	}

	return &PatchUserResponse{}, nil
}
