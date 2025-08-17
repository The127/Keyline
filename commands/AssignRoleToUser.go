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
	_, err := virtualServerRepository.First(ctx, virtualServerFilter)
	if err != nil {
		return nil, fmt.Errorf("getting virtual server: %w", err)
	}

	// TODO: check that user and role and group are all in the correct tenant (virtual server)

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
