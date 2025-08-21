package commands

import (
	"Keyline/ioc"
	"Keyline/middlewares"
	"Keyline/repositories"
	"context"
	"fmt"
	"github.com/google/uuid"
)

type AssignRoleToUser struct {
	VirtualServerName string
	UserId            uuid.UUID
	RoleId            uuid.UUID
}

type AssignRoleToUserResponse struct{}

func HandleAssignRoleToUser(ctx context.Context, command AssignRoleToUser) (*AssignRoleToUserResponse, error) {
	scope := middlewares.GetScope(ctx)

	virtualServerRepository := ioc.GetDependency[*repositories.VirtualServerRepository](scope)
	virtualServerFilter := repositories.NewVirtualServerFilter().Name(command.VirtualServerName)
	virtualServer, err := virtualServerRepository.Single(ctx, virtualServerFilter)
	if err != nil {
		return nil, fmt.Errorf("getting virtual server: %w", err)
	}

	roleRepository := ioc.GetDependency[repositories.RoleRepository](scope)
	_, err = roleRepository.Single(ctx, repositories.NewRoleFilter().Id(command.RoleId).VirtualServerId(virtualServer.Id()))
	if err != nil {
		return nil, fmt.Errorf("getting role: %w", err)
	}

	userRepository := ioc.GetDependency[*repositories.UserRepository](scope)
	_, err = userRepository.Single(ctx, repositories.NewUserFilter().Id(command.UserId).VirtualServerId(virtualServer.Id()))
	if err != nil {
		return nil, fmt.Errorf("getting user: %w", err)
	}

	userRoleAssignmentRepository := ioc.GetDependency[*repositories.UserRoleAssignmentRepository](scope)
	userRoleAssignment := repositories.NewUserRoleAssignment(
		command.UserId,
		command.RoleId,
		nil, // TODO: add group id to command once we need it
	)
	err = userRoleAssignmentRepository.Insert(ctx, userRoleAssignment)
	if err != nil {
		return nil, fmt.Errorf("inserting user role assignment: %w", err)
	}

	return &AssignRoleToUserResponse{}, nil
}
