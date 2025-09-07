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

type GetApplications struct {
	PagedQuery
	VirtualServerName string
}

type GetApplicationsResponse struct {
	PagedResponse[GetApplicationsResponseItem]
}

type GetApplicationsResponseItem struct {
	Id          uuid.UUID
	Name        string
	DisplayName string
}

func HandleGetApplications(ctx context.Context, query GetApplications) (*GetApplicationsResponse, error) {
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
		Pagination(query.Page, query.PageSize)
	applications, total, err := applicationRepository.List(ctx, applicationFilter)
	if err != nil {
		return nil, fmt.Errorf("searching applications: %w", err)
	}

	items := utils.MapSlice(applications, func(t *repositories.Application) GetApplicationsResponseItem {
		return GetApplicationsResponseItem{
			Id:          t.Id(),
			Name:        t.Name(),
			DisplayName: t.DisplayName(),
		}
	})

	return &GetApplicationsResponse{
		PagedResponse: NewPagedResponse(items, total),
	}, nil
}
