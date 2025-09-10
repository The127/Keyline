package queries

import (
	"Keyline/ioc"
	"Keyline/middlewares"
	"Keyline/repositories"
	"Keyline/utils"
	"context"
	"fmt"
	"github.com/google/uuid"
)

type ListApplications struct {
	PagedQuery
	OrderedQuery
	VirtualServerName string
	SearchText        string
}

type ListApplicationsResponse struct {
	PagedResponse[ListApplicationsResponseItem]
}

type ListApplicationsResponseItem struct {
	Id          uuid.UUID
	Name        string
	DisplayName string
}

func HandleListApplications(ctx context.Context, query ListApplications) (*ListApplicationsResponse, error) {
	scope := middlewares.GetScope(ctx)

	virtualServerRepository := ioc.GetDependency[repositories.VirtualServerRepository](scope)
	virtualServerFilter := repositories.NewVirtualServerFilter().
		Name(query.VirtualServerName)
	virtualServer, err := virtualServerRepository.Single(ctx, virtualServerFilter)
	if err != nil {
		return nil, fmt.Errorf("searching virtual servers: %w", err)
	}

	applicationRepository := ioc.GetDependency[repositories.ApplicationRepository](scope)
	applicationFilter := repositories.NewApplicationFilter().
		VirtualServerId(virtualServer.Id()).
		Pagination(query.Page, query.PageSize).
		Order(query.OrderBy, query.OrderDir).
		Search(query.SearchText)
	applications, total, err := applicationRepository.List(ctx, applicationFilter)
	if err != nil {
		return nil, fmt.Errorf("searching applications: %w", err)
	}

	items := utils.MapSlice(applications, func(t *repositories.Application) ListApplicationsResponseItem {
		return ListApplicationsResponseItem{
			Id:          t.Id(),
			Name:        t.Name(),
			DisplayName: t.DisplayName(),
		}
	})

	return &ListApplicationsResponse{
		PagedResponse: NewPagedResponse(items, total),
	}, nil
}
