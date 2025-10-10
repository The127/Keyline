package commands

import (
	"Keyline/internal/middlewares"
	repositories2 "Keyline/internal/repositories"
	"Keyline/ioc"
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

	virtualServerRepository := ioc.GetDependency[repositories2.VirtualServerRepository](scope)
	virtualServerFilter := repositories2.NewVirtualServerFilter().Name(command.VirtualServerName)
	virtualServer, err := virtualServerRepository.Single(ctx, virtualServerFilter)
	if err != nil {
		return nil, fmt.Errorf("getting virtual server: %w", err)
	}

	roleRepository := ioc.GetDependency[repositories2.RoleRepository](scope)
	_, err = roleRepository.Single(ctx, repositories2.NewRoleFilter().Id(command.RoleId).VirtualServerId(virtualServer.Id()))
	if err != nil {
		return nil, fmt.Errorf("getting role: %w", err)
	}

	userRepository := ioc.GetDependency[repositories2.UserRepository](scope)
	_, err = userRepository.Single(ctx, repositories2.NewUserFilter().Id(command.UserId).VirtualServerId(virtualServer.Id()))
	if err != nil {
		return nil, fmt.Errorf("getting user: %w", err)
	}

	userRoleAssignmentRepository := ioc.GetDependency[repositories2.UserRoleAssignmentRepository](scope)
	userRoleAssignment := repositories2.NewUserRoleAssignment(
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
