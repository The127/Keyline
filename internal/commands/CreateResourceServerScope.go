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

type CreateResourceServerScope struct {
	VirtualServerName string
	ProjectSlug       string
	ResourceServerId  uuid.UUID
	Scope             string
	Name              string
	Description       string
}

func (a CreateResourceServerScope) LogRequest() bool {
	return true
}

func (a CreateResourceServerScope) LogResponse() bool {
	return true
}

func (a CreateResourceServerScope) IsAllowed(ctx context.Context) (behaviours.PolicyResult, error) {
	return behaviours.PermissionBasedPolicy(ctx, permissions.ResourceServerScopeCreate)
}

type CreateResourceServerScopeResponse struct {
	Id uuid.UUID
}

func HandleCreateResourceServerScope(ctx context.Context, command CreateResourceServerScope) (*CreateResourceServerScopeResponse, error) {
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
	resourceServer, err := dbContext.ResourceServers().FirstOrErr(ctx, resourceServerFilter)
	if err != nil {
		return nil, fmt.Errorf("getting resource server: %w", err)
	}

	resourceServerScope := repositories.NewResourceServerScope(virtualServer.Id(), project.Id(), resourceServer.Id(), command.Scope, command.Name)
	resourceServerScope.SetDescription(command.Description)
	dbContext.ResourceServerScopes().Insert(resourceServerScope)

	return &CreateResourceServerScopeResponse{
		Id: resourceServerScope.Id(),
	}, nil
}
