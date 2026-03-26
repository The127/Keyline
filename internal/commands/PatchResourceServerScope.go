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

type PatchResourceServerScope struct {
	VirtualServerName string
	ProjectSlug       string
	ResourceServerId  uuid.UUID
	ScopeId           uuid.UUID
	Name              *string
	Description       *string
}

func (a PatchResourceServerScope) LogRequest() bool {
	return true
}

func (a PatchResourceServerScope) LogResponse() bool {
	return true
}

func (a PatchResourceServerScope) IsAllowed(ctx context.Context) (behaviours.PolicyResult, error) {
	return behaviours.PermissionBasedPolicy(ctx, permissions.ResourceServerScopeUpdate)
}

func (a PatchResourceServerScope) GetRequestName() string {
	return "PatchResourceServerScope"
}

type PatchResourceServerScopeResponse struct{}

func HandlePatchResourceServerScope(ctx context.Context, command PatchResourceServerScope) (*PatchResourceServerScopeResponse, error) {
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
	rssScope, err := dbContext.ResourceServerScopes().FirstOrErr(ctx, scopeFilter)
	if err != nil {
		return nil, fmt.Errorf("getting resource server scope: %w", err)
	}

	if command.Name != nil {
		rssScope.SetName(*command.Name)
	}
	if command.Description != nil {
		rssScope.SetDescription(*command.Description)
	}

	dbContext.ResourceServerScopes().Update(rssScope)
	return &PatchResourceServerScopeResponse{}, nil
}
