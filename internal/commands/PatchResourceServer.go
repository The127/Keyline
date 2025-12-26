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

	resourceServerFilter := repositories.NewResourceServerFilter().
		VirtualServerId(virtualServer.Id()).
		ProjectId(project.Id()).
		Id(command.ResourceServerId)
	resourceServer, err := dbContext.ResourceServers().Single(ctx, resourceServerFilter)
	if err != nil {
		return nil, fmt.Errorf("getting resource server: %w", err)
	}

	if command.Name != nil {
		resourceServer.SetName(*command.Name)
	}
	if command.Description != nil {
		resourceServer.SetDescription(*command.Description)
	}

	dbContext.ResourceServers().Update(resourceServer)
	return &PatchResourceServerResponse{}, nil
}
