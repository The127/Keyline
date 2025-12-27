package queries

import (
	"Keyline/internal/authentication/permissions"
	"Keyline/internal/behaviours"
	"Keyline/internal/database"
	"Keyline/internal/middlewares"
	"Keyline/internal/repositories"
	"context"
	"fmt"
	"time"

	"github.com/The127/ioc"

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
	dbContext := ioc.GetDependency[database.Context](scope)

	virtualServerFilter := repositories.NewVirtualServerFilter().Name(query.VirtualServerName)
	virtualServer, err := dbContext.VirtualServers().FirstOrErr(ctx, virtualServerFilter)
	if err != nil {
		return nil, fmt.Errorf("getting virtual server: %w", err)
	}

	projectFilter := repositories.NewProjectFilter().VirtualServerId(virtualServer.Id()).Slug(query.ProjectSlug)
	project, err := dbContext.Projects().FirstOrErr(ctx, projectFilter)
	if err != nil {
		return nil, fmt.Errorf("getting project: %w", err)
	}

	resourceServerScopeFilter := repositories.NewResourceServerScopeFilter().
		VirtualServerId(virtualServer.Id()).
		ProjectId(project.Id()).
		ResourceServerId(query.ResourceServerId).
		Id(query.ScopeId)
	resourceServerScope, err := dbContext.ResourceServerScopes().FirstOrErr(ctx, resourceServerScopeFilter)
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
