package commands

import (
	"Keyline/internal/authentication/permissions"
	"Keyline/internal/behaviours"
	"Keyline/internal/database"
	"Keyline/internal/middlewares"
	"Keyline/internal/repositories"
	"context"
	"fmt"

	"github.com/The127/ioc"

	"github.com/google/uuid"
)

type AssignRoleToUser struct {
	VirtualServerName string
	ProjectSlug       string
	UserId            uuid.UUID
	RoleId            uuid.UUID
}

func (a AssignRoleToUser) LogRequest() bool {
	return true
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
	dbContext := ioc.GetDependency[database.Context](scope)

	virtualServerFilter := repositories.NewVirtualServerFilter().Name(command.VirtualServerName)
	virtualServer, err := dbContext.VirtualServers().Single(ctx, virtualServerFilter)
	if err != nil {
		return nil, fmt.Errorf("getting virtual server: %w", err)
	}

	projectFilter := repositories.NewProjectFilter().VirtualServerId(virtualServer.Id()).Slug(command.ProjectSlug)
	project, err := dbContext.Projects().Single(ctx, projectFilter)
	if err != nil {
		return nil, fmt.Errorf("getting project: %w", err)
	}

	roleFilter := repositories.NewRoleFilter().
		Id(command.RoleId).
		VirtualServerId(virtualServer.Id()).
		ProjectId(project.Id())

	_, err = dbContext.Roles().Single(ctx, roleFilter)
	if err != nil {
		return nil, fmt.Errorf("getting role: %w", err)
	}

	_, err = dbContext.Users().Single(ctx, repositories.NewUserFilter().Id(command.UserId).VirtualServerId(virtualServer.Id()))
	if err != nil {
		return nil, fmt.Errorf("getting user: %w", err)
	}

	userRoleAssignment := repositories.NewUserRoleAssignment(command.UserId, command.RoleId, nil)
	dbContext.UserRoleAssignments().Insert(userRoleAssignment)

	return &AssignRoleToUserResponse{}, nil
}
