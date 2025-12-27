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
	dbContext := ioc.GetDependency[database.Context](scope)

	virtualServerFilter := repositories.NewVirtualServerFilter().Name(command.VirtualServerName)
	virtualServer, err := dbContext.VirtualServers().FirstOrErr(ctx, virtualServerFilter)
	if err != nil {
		return nil, fmt.Errorf("getting virtual server: %w", err)
	}

	projectFilter := repositories.NewProjectFilter().VirtualServerId(virtualServer.Id()).Slug(command.Slug)
	project, err := dbContext.Projects().FirstOrErr(ctx, projectFilter)
	if err != nil {
		return nil, fmt.Errorf("getting project: %w", err)
	}

	if command.Name != nil {
		project.SetName(*command.Name)
	}
	if command.Description != nil {
		project.SetDescription(*command.Description)
	}

	dbContext.Projects().Update(project)
	return &PatchProjectResponse{}, nil
}
