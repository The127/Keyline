package commands

import (
	"Keyline/internal/authentication/permissions"
	"Keyline/internal/behaviours"
	"Keyline/internal/middlewares"
	"Keyline/internal/repositories"
	"Keyline/ioc"
	"context"
	"fmt"

	"github.com/google/uuid"
)

type AssignRoleToUser struct {
	VirtualServerName string
	UserId            uuid.UUID
	RoleId            uuid.UUID
	ApplicationId     *uuid.UUID
}

func (a AssignRoleToUser) LogResponse() bool {
	return true
}

func (a AssignRoleToUser) IsAllowed(ctx context.Context) (behaviours.PolicyResult, error) {
	return behaviours.PermissionBasedPolicy(ctx, permissions.RoleAssign)
}

func (a AssignRoleToUser) GetRequestName() string {
	return "AssignRoleToUser"
}

type AssignRoleToUserResponse struct{}

func HandleAssignRoleToUser(ctx context.Context, command AssignRoleToUser) (*AssignRoleToUserResponse, error) {
	scope := middlewares.GetScope(ctx)

	virtualServerRepository := ioc.GetDependency[repositories.VirtualServerRepository](scope)
	virtualServerFilter := repositories.NewVirtualServerFilter().Name(command.VirtualServerName)
	virtualServer, err := virtualServerRepository.Single(ctx, virtualServerFilter)
	if err != nil {
		return nil, fmt.Errorf("getting virtual server: %w", err)
	}

	roleRepository := ioc.GetDependency[repositories.RoleRepository](scope)

	roleFilter := repositories.NewRoleFilter().
		Id(command.RoleId).
		VirtualServerId(virtualServer.Id())

	if command.ApplicationId != nil {
		roleFilter = roleFilter.ApplicationId(*command.ApplicationId)
	}

	_, err = roleRepository.Single(ctx, roleFilter)
	if err != nil {
		return nil, fmt.Errorf("getting role: %w", err)
	}

	userRepository := ioc.GetDependency[repositories.UserRepository](scope)
	_, err = userRepository.Single(ctx, repositories.NewUserFilter().Id(command.UserId).VirtualServerId(virtualServer.Id()))
	if err != nil {
		return nil, fmt.Errorf("getting user: %w", err)
	}

	userRoleAssignmentRepository := ioc.GetDependency[repositories.UserRoleAssignmentRepository](scope)
	userRoleAssignment := repositories.NewUserRoleAssignment(
		command.UserId,
		command.RoleId,
		nil, // TODO: add group id to command once we need it
		command.ApplicationId,
	)
	err = userRoleAssignmentRepository.Insert(ctx, userRoleAssignment)
	if err != nil {
		return nil, fmt.Errorf("inserting user role assignment: %w", err)
	}

	return &AssignRoleToUserResponse{}, nil
}
