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

type PatchRole struct {
	VirtualServerName string
	ProjectSlug       string
	RoleId            uuid.UUID
	Name              *string
	Description       *string
}

func (a PatchRole) LogRequest() bool {
	return true
}

func (a PatchRole) LogResponse() bool {
	return true
}

func (a PatchRole) IsAllowed(ctx context.Context) (behaviours.PolicyResult, error) {
	return behaviours.PermissionBasedPolicy(ctx, permissions.RoleUpdate)
}

func (a PatchRole) GetRequestName() string {
	return "PatchRole"
}

type PatchRoleResponse struct{}

func HandlePatchRole(ctx context.Context, command PatchRole) (*PatchRoleResponse, error) {
	scope := middlewares.GetScope(ctx)
	dbContext := ioc.GetDependency[database.Context](scope)

	virtualServerFilter := repositories.NewVirtualServerFilter().Name(command.VirtualServerName)
	virtualServer, err := dbContext.VirtualServers().Single(ctx, virtualServerFilter)
	if err != nil {
		return nil, fmt.Errorf("getting virtual server: %w", err)
	}

	projectFilter := repositories.NewProjectFilter().
		VirtualServerId(virtualServer.Id()).
		Slug(command.ProjectSlug)
	project, err := dbContext.Projects().Single(ctx, projectFilter)
	if err != nil {
		return nil, fmt.Errorf("getting project: %w", err)
	}

	roleFilter := repositories.NewRoleFilter().
		VirtualServerId(virtualServer.Id()).
		ProjectId(project.Id()).
		Id(command.RoleId)
	role, err := dbContext.Roles().Single(ctx, roleFilter)
	if err != nil {
		return nil, fmt.Errorf("getting role: %w", err)
	}

	if command.Name != nil {
		role.SetName(*command.Name)
	}
	if command.Description != nil {
		role.SetDescription(*command.Description)
	}

	dbContext.Roles().Update(role)
	return &PatchRoleResponse{}, nil
}
