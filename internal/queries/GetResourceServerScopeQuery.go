package queries

import (
	"Keyline/internal/authentication/permissions"
	"Keyline/internal/behaviours"
	"Keyline/internal/middlewares"
	"Keyline/internal/repositories"
	"Keyline/ioc"
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type GetResourceServerScope struct {
	VirtualServerName string
	ProjectSlug       string
	ResourceServerId  uuid.UUID
	ScopeId           uuid.UUID
}

func (a GetResourceServerScope) LogRequest() bool {
	return true
}

func (a GetResourceServerScope) LogResponse() bool {
	return false
}

func (a GetResourceServerScope) IsAllowed(ctx context.Context) (behaviours.PolicyResult, error) {
	return behaviours.PermissionBasedPolicy(ctx, permissions.ResourceServerScopeCreate)
}

func (a GetResourceServerScope) GetRequestName() string {
	return "GetResourceServerScope"
}

type GetResourceServerScopeResponse struct {
	Id          uuid.UUID
	Scope       string
	Name        string
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func HandleGetResourceServerScope(ctx context.Context, query GetResourceServerScope) (*GetResourceServerScopeResponse, error) {
	scope := middlewares.GetScope(ctx)

	virtualServerRepository := ioc.GetDependency[repositories.VirtualServerRepository](scope)
	virtualServerFilter := repositories.NewVirtualServerFilter().Name(query.VirtualServerName)
	virtualServer, err := virtualServerRepository.Single(ctx, virtualServerFilter)
	if err != nil {
		return nil, fmt.Errorf("getting virtual server: %w", err)
	}

	projectRepository := ioc.GetDependency[repositories.ProjectRepository](scope)
	projectFilter := repositories.NewProjectFilter().VirtualServerId(virtualServer.Id()).Slug(query.ProjectSlug)
	project, err := projectRepository.Single(ctx, projectFilter)
	if err != nil {
		return nil, fmt.Errorf("getting project: %w", err)
	}
	resourceServerScopeRepository := ioc.GetDependency[repositories.ResourceServerScopeRepository](scope)
	resourceServerScopeFilter := repositories.NewResourceServerScopeFilter().
		VirtualServerId(virtualServer.Id()).
		ProjectId(project.Id()).
		ResourceServerId(query.ResourceServerId).
		Id(query.ScopeId)
	resourceServerScope, err := resourceServerScopeRepository.Single(ctx, resourceServerScopeFilter)
	if err != nil {
		return nil, fmt.Errorf("getting resource server scope: %w", err)
	}

	return &GetResourceServerScopeResponse{
		Id:          resourceServerScope.Id(),
		Scope:       resourceServerScope.Scope(),
		Name:        resourceServerScope.Name(),
		Description: resourceServerScope.Description(),
		CreatedAt:   resourceServerScope.AuditUpdatedAt(),
		UpdatedAt:   resourceServerScope.AuditCreatedAt(),
	}, nil
}
