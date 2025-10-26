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

	resourceServerScopeRepository := ioc.GetDependency[repositories.ResourceServerScopeRepository](scope)
	resourceServerScope := repositories.NewResourceServerScope(virtualServer.Id(), project.Id(), resourceServer.Id(), command.Scope, command.Name)
	resourceServerScope.SetDescription(command.Description)
	err = resourceServerScopeRepository.Insert(ctx, resourceServerScope)
	if err != nil {
		return nil, fmt.Errorf("inserting resource server scope: %w", err)
	}

	return &CreateResourceServerScopeResponse{
		Id: resourceServerScope.Id(),
	}, nil
}
