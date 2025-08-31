package queries

import (
	"Keyline/ioc"
	"Keyline/middlewares"
	"Keyline/repositories"
	"Keyline/utils"
	"context"
	"fmt"
)

type GetApplications struct {
	VirtualServerName string
}

type GetApplicationsResponse struct {
	Name        string
	DisplayName string
}

func HandleGetApplications(ctx context.Context, _ GetApplications) ([]GetApplicationsResponse, error) {
	scope := middlewares.GetScope(ctx)

	virtualServerRepository := ioc.GetDependency[repositories.VirtualServerRepository](scope)
	virtualServer, err := virtualServerRepository.Single(ctx, repositories.NewVirtualServerFilter())
	if err != nil {
		return nil, fmt.Errorf("searching virtual servers: %w", err)
	}

	applicationRepository := ioc.GetDependency[repositories.ApplicationRepository](scope)
	applicationFilter := repositories.NewApplicationFilter().
		VirtualServerId(virtualServer.Id())
	applications, err := applicationRepository.List(ctx, applicationFilter)
	if err != nil {
		return nil, fmt.Errorf("searching applications: %w", err)
	}

	result := utils.MapSlice(applications, func(t *repositories.Application) GetApplicationsResponse {
		return GetApplicationsResponse{
			Name:        t.Name(),
			DisplayName: t.DisplayName(),
		}
	})

	return result, nil
}
