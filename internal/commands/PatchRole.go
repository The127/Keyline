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

	virtualServerRepository := ioc.GetDependency[repositories.VirtualServerRepository](scope)
	virtualServerFilter := repositories.NewVirtualServerFilter().Name(command.VirtualServerName)
	virtualServer, err := virtualServerRepository.Single(ctx, virtualServerFilter)
	if err != nil {
		return nil, fmt.Errorf("getting virtual server: %w", err)
	}

	projectRepository := ioc.GetDependency[repositories.ProjectRepository](scope)
	projectFilter := repositories.NewProjectFilter().
		VirtualServerId(virtualServer.Id()).
		Slug(command.ProjectSlug)
	project, err := projectRepository.Single(ctx, projectFilter)
	if err != nil {
		return nil, fmt.Errorf("getting project: %w", err)
	}

	roleRepository := ioc.GetDependency[repositories.RoleRepository](scope)
	roleFilter := repositories.NewRoleFilter().
		VirtualServerId(virtualServer.Id()).
		ProjectId(project.Id()).
		Id(command.RoleId)
	role, err := roleRepository.Single(ctx, roleFilter)
	if err != nil {
		return nil, fmt.Errorf("getting role: %w", err)
	}

	if command.Name != nil {
		role.SetName(*command.Name)
	}
	if command.Description != nil {
		role.SetDescription(*command.Description)
	}

	err = roleRepository.Update(ctx, role)
	if err != nil {
		return nil, fmt.Errorf("updating role: %w", err)
	}

	return &PatchRoleResponse{}, nil
}
