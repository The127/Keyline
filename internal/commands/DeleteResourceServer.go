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

type DeleteResourceServer struct {
	VirtualServerName string
	ProjectSlug       string
	ResourceServerId  uuid.UUID
}

func (a DeleteResourceServer) LogRequest() bool {
	return true
}

func (a DeleteResourceServer) LogResponse() bool {
	return true
}

func (a DeleteResourceServer) IsAllowed(ctx context.Context) (behaviours.PolicyResult, error) {
	return behaviours.PermissionBasedPolicy(ctx, permissions.ResourceServerDelete)
}

func (a DeleteResourceServer) GetRequestName() string {
	return "DeleteResourceServer"
}

type DeleteResourceServerResponse struct{}

func HandleDeleteResourceServer(ctx context.Context, command DeleteResourceServer) (*DeleteResourceServerResponse, error) {
	scope := middlewares.GetScope(ctx)
	dbContext := ioc.GetDependency[database.Context](scope)

	virtualServerFilter := repositories.NewVirtualServerFilter().Name(command.VirtualServerName)
	virtualServer, err := dbContext.VirtualServers().FirstOrErr(ctx, virtualServerFilter)
	if err != nil {
		return nil, fmt.Errorf("getting virtual server: %w", err)
	}

	projectFilter := repositories.NewProjectFilter().VirtualServerId(virtualServer.Id()).Slug(command.ProjectSlug)
	project, err := dbContext.Projects().FirstOrErr(ctx, projectFilter)
	if err != nil {
		return nil, fmt.Errorf("getting project: %w", err)
	}

	resourceServerFilter := repositories.NewResourceServerFilter().
		VirtualServerId(virtualServer.Id()).
		ProjectId(project.Id()).
		Id(command.ResourceServerId)
	resourceServer, err := dbContext.ResourceServers().FirstOrNil(ctx, resourceServerFilter)
	if err != nil {
		return nil, fmt.Errorf("getting resource server: %w", err)
	}

	if resourceServer == nil {
		return &DeleteResourceServerResponse{}, nil
	}

	dbContext.ResourceServers().Delete(resourceServer.Id())

	return &DeleteResourceServerResponse{}, nil
}
