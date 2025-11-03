package queries

import (
	"Keyline/internal/authentication/permissions"
	"Keyline/internal/behaviours"
	"Keyline/internal/middlewares"
	"Keyline/internal/repositories"
	"Keyline/ioc"
	"Keyline/utils"
	"context"
	"fmt"

	"github.com/google/uuid"
)

type ListRessouceServerScopes struct {
	PagedQuery
	OrderedQuery
	VirtualServerName string
	ProjectSlug       string
	ResourceServerId  uuid.UUID
	SearchText        string
}

func (a ListRessouceServerScopes) LogRequest() bool {
	return true
}

func (a ListRessouceServerScopes) LogResponse() bool {
	return false
}

func (a ListRessouceServerScopes) IsAllowed(ctx context.Context) (behaviours.PolicyResult, error) {
	return behaviours.PermissionBasedPolicy(ctx, permissions.ResourceServerScopeCreate)
}

func (a ListRessouceServerScopes) GetRequestName() string {
	return "ListRessouceServerScopes"
}

type ListResourceServerScopesResponse struct {
	PagedResponse[ListResourceServerScopesResponseItem]
}

type ListResourceServerScopesResponseItem struct {
	Id    uuid.UUID
	Name  string
	Scope string
}

func HandleListResourceServerScopes(ctx context.Context, query ListRessouceServerScopes) (*ListResourceServerScopesResponse, error) {
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

	resourceServerRepository := ioc.GetDependency[repositories.ResourceServerRepository](scope)
	resourceServerFilter := repositories.NewResourceServerFilter().
		VirtualServerId(virtualServer.Id()).
		ProjectId(project.Id()).
		Id(query.ResourceServerId)
	resourceServer, err := resourceServerRepository.Single(ctx, resourceServerFilter)
	if err != nil {
		return nil, fmt.Errorf("getting resource server: %w", err)
	}

	resourceServerScopeRepository := ioc.GetDependency[repositories.ResourceServerScopeRepository](scope)
	resourceServerScopeFilter := repositories.NewResourceServerScopeFilter().
		VirtualServerId(virtualServer.Id()).
		ProjectId(project.Id()).
		ResourceServerId(resourceServer.Id()).
		Pagination(query.Page, query.PageSize).
		Order(query.OrderBy, query.OrderDir).
		Search(repositories.NewContainsSearchFilter(query.SearchText))
	resourceServerScopes, total, err := resourceServerScopeRepository.List(ctx, resourceServerScopeFilter)
	if err != nil {
		return nil, fmt.Errorf("getting resource server scopes: %w", err)
	}

	items := utils.MapSlice(resourceServerScopes, func(t *repositories.ResourceServerScope) ListResourceServerScopesResponseItem {
		return ListResourceServerScopesResponseItem{
			Id:    t.Id(),
			Name:  t.Name(),
			Scope: t.Scope(),
		}
	})

	return &ListResourceServerScopesResponse{
		PagedResponse: NewPagedResponse(items, total),
	}, nil
}
