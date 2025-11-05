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

type CreateResourceServer struct {
	VirtualServerName string
	ProjectSlug       string
	Slug              string
	Name              string
	Description       string
}

func (a CreateResourceServer) LogRequest() bool {
	return true
}

func (a CreateResourceServer) LogResponse() bool {
	return true
}

func (a CreateResourceServer) IsAllowed(ctx context.Context) (behaviours.PolicyResult, error) {
	return behaviours.PermissionBasedPolicy(ctx, permissions.ResourceServerCreate)
}

func (a CreateResourceServer) GetRequestName() string {
	return "CreateResourceServer"
}

type CreateResourceServerResponse struct {
	Id uuid.UUID
}

func HandleCreateResourceServer(ctx context.Context, command CreateResourceServer) (*CreateResourceServerResponse, error) {
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
	resourceServer := repositories.NewResourceServer(virtualServer.Id(), project.Id(), command.Slug, command.Name, command.Description)
	err = resourceServerRepository.Insert(ctx, resourceServer)
	if err != nil {
		return nil, fmt.Errorf("inserting resource server: %w", err)
	}

	return &CreateResourceServerResponse{
		Id: resourceServer.Id(),
	}, nil
}
