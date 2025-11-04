package commands

import (
	"Keyline/internal/authentication/permissions"
	"Keyline/internal/behaviours"
	"Keyline/internal/middlewares"
	"Keyline/internal/repositories"
	"Keyline/ioc"
	"context"
	"fmt"
)

type PatchProject struct {
	VirtualServerName string
	Slug              string
	Name              *string
	Description       *string
}

func (a PatchProject) LogRequest() bool {
	return true
}

func (a PatchProject) LogResponse() bool {
	return true
}

func (a PatchProject) IsAllowed(ctx context.Context) (behaviours.PolicyResult, error) {
	return behaviours.PermissionBasedPolicy(ctx, permissions.ProjectUpdate)
}

func (a PatchProject) GetRequestName() string {
	return "PatchProject"
}

type PatchProjectResponse struct{}

func HandlePatchProject(ctx context.Context, command PatchProject) (*PatchProjectResponse, error) {
	scope := middlewares.GetScope(ctx)

	virtualServerRepository := ioc.GetDependency[repositories.VirtualServerRepository](scope)
	virtualServerFilter := repositories.NewVirtualServerFilter().Name(command.VirtualServerName)
	virtualServer, err := virtualServerRepository.Single(ctx, virtualServerFilter)
	if err != nil {
		return nil, fmt.Errorf("getting virtual server: %w", err)
	}

	projectRepository := ioc.GetDependency[repositories.ProjectRepository](scope)
	projectFilter := repositories.NewProjectFilter().VirtualServerId(virtualServer.Id()).Slug(command.Slug)
	project, err := projectRepository.Single(ctx, projectFilter)
	if err != nil {
		return nil, fmt.Errorf("getting project: %w", err)
	}

	if command.Name != nil {
		project.SetName(*command.Name)
	}
	if command.Description != nil {
		project.SetDescription(*command.Description)
	}

	err = projectRepository.Update(ctx, project)
	if err != nil {
		return nil, fmt.Errorf("updating project: %w", err)
	}

	return &PatchProjectResponse{}, nil
}
