package queries

import (
	"Keyline/internal/authentication/permissions"
	"Keyline/internal/behaviours"
	"Keyline/internal/middlewares"
	"Keyline/internal/repositories"
	"Keyline/utils"
	"context"
	"fmt"

	"github.com/The127/ioc"

	"github.com/google/uuid"
)

type ListProjects struct {
	PagedQuery
	OrderedQuery
	VirtualServerName string
	SearchText        string
}

func (a ListProjects) LogRequest() bool {
	return true
}

func (a ListProjects) LogResponse() bool {
	return false
}

func (a ListProjects) IsAllowed(ctx context.Context) (behaviours.PolicyResult, error) {
	return behaviours.PermissionBasedPolicy(ctx, permissions.ProjectView)
}

func (a ListProjects) GetRequestName() string {
	return "ListProjects"
}

type ListProjectsResponse struct {
	PagedResponse[ListProjectsResponseItem]
}

type ListProjectsResponseItem struct {
	Id            uuid.UUID
	Slug          string
	Name          string
	SystemProject bool
}

func HandleListProjects(ctx context.Context, query ListProjects) (*ListProjectsResponse, error) {
	scope := middlewares.GetScope(ctx)

	virtualServerRepository := ioc.GetDependency[repositories.VirtualServerRepository](scope)
	virtualServerFilter := repositories.NewVirtualServerFilter().Name(query.VirtualServerName)
	virtualServer, err := virtualServerRepository.Single(ctx, virtualServerFilter)
	if err != nil {
		return nil, fmt.Errorf("getting virtual server: %w", err)
	}

	projectRepository := ioc.GetDependency[repositories.ProjectRepository](scope)
	projectFilter := repositories.NewProjectFilter().
		VirtualServerId(virtualServer.Id()).
		Pagination(query.Page, query.PageSize).
		Order(query.OrderBy, query.OrderDir).
		Search(repositories.NewContainsSearchFilter(query.SearchText))
	projects, total, err := projectRepository.List(ctx, projectFilter)
	if err != nil {
		return nil, fmt.Errorf("getting projects: %w", err)
	}

	items := utils.MapSlice(projects, func(t *repositories.Project) ListProjectsResponseItem {
		return ListProjectsResponseItem{
			Id:            t.Id(),
			Slug:          t.Slug(),
			Name:          t.Name(),
			SystemProject: t.SystemProject(),
		}
	})

	return &ListProjectsResponse{
		PagedResponse: NewPagedResponse(items, total),
	}, nil
}
