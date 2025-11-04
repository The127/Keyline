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

type PatchResourceServer struct {
	VirtualServerName string
	ProjectSlug       string
	ResourceServerId  uuid.UUID
	Name              *string
	Description       *string
}

func (a PatchResourceServer) LogRequest() bool {
	return true
}

func (a PatchResourceServer) LogResponse() bool {
	return true
}

func (a PatchResourceServer) IsAllowed(ctx context.Context) (behaviours.PolicyResult, error) {
	return behaviours.PermissionBasedPolicy(ctx, permissions.ResourceServerUpdate)
}

func (a PatchResourceServer) GetRequestName() string {
	return "PatchResourceServer"
}

type PatchResourceServerResponse struct{}

func HandlePatchResourceServer(ctx context.Context, command PatchResourceServer) (*PatchResourceServerResponse, error) {
	scope := middlewares.GetScope(ctx)

	virtualServerRepository := ioc.GetDependency[repositories.VirtualServerRepository](scope)
	virtualServerFilter := repositories.NewVirtualServerFilter().Name(command.VirtualServerName)
	virtualServer, err := virtualServerRepository.Single(ctx, virtualServerFilter)
	if err != nil {
		return nil, fmt.Errorf("getting virtual server: %w", err)
	}

	projectRepository := ioc.GetDependency[repositories.ProjectRepository](scope)
	projectFilter := repositories.NewProjectFilter().VirtualServerId(virtualServer.Id()).Slug(command.ProjectSlug)
	project, err := projectRepository.Single(ctx, projectFilter)
	if err != nil {
		return nil, fmt.Errorf("getting project: %w", err)
	}

	resourceServerRepository := ioc.GetDependency[repositories.ResourceServerRepository](scope)
	resourceServerFilter := repositories.NewResourceServerFilter().
		VirtualServerId(virtualServer.Id()).
		ProjectId(project.Id()).
		Id(command.ResourceServerId)
	resourceServer, err := resourceServerRepository.Single(ctx, resourceServerFilter)
	if err != nil {
		return nil, fmt.Errorf("getting resource server: %w", err)
	}

	if command.Name != nil {
		resourceServer.SetName(*command.Name)
	}
	if command.Description != nil {
		resourceServer.SetDescription(*command.Description)
	}

	err = resourceServerRepository.Update(ctx, resourceServer)
	if err != nil {
		return nil, fmt.Errorf("updating resource server: %w", err)
	}

	return &PatchResourceServerResponse{}, nil
}
