package queries

import (
	"Keyline/internal/middlewares"
	repositories2 "Keyline/internal/repositories"
	"Keyline/ioc"
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
	Type        repositories2.ApplicationType
}

func HandleListApplications(ctx context.Context, query ListApplications) (*ListApplicationsResponse, error) {
	scope := middlewares.GetScope(ctx)

	virtualServerRepository := ioc.GetDependency[repositories2.VirtualServerRepository](scope)
	virtualServerFilter := repositories2.NewVirtualServerFilter().
		Name(query.VirtualServerName)
	virtualServer, err := virtualServerRepository.Single(ctx, virtualServerFilter)
	if err != nil {
		return nil, fmt.Errorf("searching virtual servers: %w", err)
	}

	applicationRepository := ioc.GetDependency[repositories2.ApplicationRepository](scope)
	applicationFilter := repositories2.NewApplicationFilter().
		VirtualServerId(virtualServer.Id()).
		Pagination(query.Page, query.PageSize).
		Order(query.OrderBy, query.OrderDir).
		Search(query.SearchText)
	applications, total, err := applicationRepository.List(ctx, applicationFilter)
	if err != nil {
		return nil, fmt.Errorf("searching applications: %w", err)
	}

	items := utils.MapSlice(applications, func(t *repositories2.Application) ListApplicationsResponseItem {
		return ListApplicationsResponseItem{
			Id:          t.Id(),
			Name:        t.Name(),
			DisplayName: t.DisplayName(),
			Type:        t.Type(),
		}
	})

	return &ListApplicationsResponse{
		PagedResponse: NewPagedResponse(items, total),
	}, nil
}
