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

type DeleteResourceServerScope struct {
	VirtualServerName string
	ProjectSlug       string
	ResourceServerId  uuid.UUID
	ScopeId           uuid.UUID
}

func (a DeleteResourceServerScope) LogRequest() bool {
	return true
}

func (a DeleteResourceServerScope) LogResponse() bool {
	return true
}

func (a DeleteResourceServerScope) IsAllowed(ctx context.Context) (behaviours.PolicyResult, error) {
	return behaviours.PermissionBasedPolicy(ctx, permissions.ResourceServerScopeDelete)
}

func (a DeleteResourceServerScope) GetRequestName() string {
	return "DeleteResourceServerScope"
}

type DeleteResourceServerScopeResponse struct{}

func HandleDeleteResourceServerScope(ctx context.Context, command DeleteResourceServerScope) (*DeleteResourceServerScopeResponse, error) {
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

	scopeFilter := repositories.NewResourceServerScopeFilter().
		VirtualServerId(virtualServer.Id()).
		ProjectId(project.Id()).
		ResourceServerId(command.ResourceServerId).
		Id(command.ScopeId)
	rssScope, err := dbContext.ResourceServerScopes().FirstOrNil(ctx, scopeFilter)
	if err != nil {
		return nil, fmt.Errorf("getting resource server scope: %w", err)
	}

	if rssScope == nil {
		return &DeleteResourceServerScopeResponse{}, nil
	}

	dbContext.ResourceServerScopes().Delete(rssScope.Id())

	return &DeleteResourceServerScopeResponse{}, nil
}
