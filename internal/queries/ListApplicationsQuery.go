package queries

import (
	"Keyline/internal/authentication/permissions"
	"Keyline/internal/behaviours"
	"Keyline/internal/database"
	"Keyline/internal/middlewares"
	"Keyline/internal/repositories"
	"Keyline/utils"
	"context"
	"fmt"

	"github.com/The127/ioc"

	"github.com/google/uuid"
)

type ListApplications struct {
	PagedQuery
	OrderedQuery
	VirtualServerName string
	ProjectSlug       string
	SearchText        string
}

func (a ListApplications) LogRequest() bool {
	return true
}

func (a ListApplications) LogResponse() bool {
	return false
}

func (a ListApplications) IsAllowed(ctx context.Context) (behaviours.PolicyResult, error) {
	return behaviours.PermissionBasedPolicy(ctx, permissions.ApplicationView)
}

func (a ListApplications) GetRequestName() string {
	return "ListApplications"
}

type ListApplicationsResponse struct {
	PagedResponse[ListApplicationsResponseItem]
}

type ListApplicationsResponseItem struct {
	Id                uuid.UUID
	Name              string
	DisplayName       string
	Type              repositories.ApplicationType
	SystemApplication bool
}

func HandleListApplications(ctx context.Context, query ListApplications) (*ListApplicationsResponse, error) {
	scope := middlewares.GetScope(ctx)
	dbContext := ioc.GetDependency[database.Context](scope)

	virtualServerFilter := repositories.NewVirtualServerFilter().
		Name(query.VirtualServerName)
	virtualServer, err := dbContext.VirtualServers().FirstOrErr(ctx, virtualServerFilter)
	if err != nil {
		return nil, fmt.Errorf("searching virtual servers: %w", err)
	}

	projectFilter := repositories.NewProjectFilter().
		VirtualServerId(virtualServer.Id()).
		Slug(query.ProjectSlug)
	project, err := dbContext.Projects().FirstOrErr(ctx, projectFilter)
	if err != nil {
		return nil, fmt.Errorf("getting project: %w", err)
	}

	applicationFilter := repositories.NewApplicationFilter().
		VirtualServerId(virtualServer.Id()).
		ProjectId(project.Id()).
		Pagination(query.Page, query.PageSize).
		Order(query.OrderBy, query.OrderDir).
		Search(repositories.NewContainsSearchFilter(query.SearchText))
	applications, total, err := dbContext.Applications().List(ctx, applicationFilter)
	if err != nil {
		return nil, fmt.Errorf("searching applications: %w", err)
	}

	items := utils.MapSlice(applications, func(t *repositories.Application) ListApplicationsResponseItem {
		return ListApplicationsResponseItem{
			Id:                t.Id(),
			Name:              t.Name(),
			DisplayName:       t.DisplayName(),
			Type:              t.Type(),
			SystemApplication: t.SystemApplication(),
		}
	})

	return &ListApplicationsResponse{
		PagedResponse: NewPagedResponse(items, total),
	}, nil
}
