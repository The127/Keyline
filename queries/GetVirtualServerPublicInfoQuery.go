package queries

import (
	"Keyline/ioc"
	"Keyline/middlewares"
	"Keyline/repositories"
	"context"
	"fmt"
)

type GetVirtualServerPublicInfo struct {
	VirtualServerName string
}

type GetVirtualServerPublicInfoResponse struct {
	Name                string
	DisplayName         string
	RegistrationEnabled bool
}

func HandleGetVirtualServerPublicInfo(ctx context.Context, query GetVirtualServerPublicInfo) (*GetVirtualServerPublicInfoResponse, error) {
	scope := middlewares.GetScope(ctx)

	virtualServerRepository := ioc.GetDependency[repositories.VirtualServerRepository](scope)
	virtualServer, err := virtualServerRepository.Single(ctx, repositories.NewVirtualServerFilter())
	if err != nil {
		return nil, fmt.Errorf("searching virtual servers: %w", err)
	}

	return &GetVirtualServerPublicInfoResponse{
		Name:                virtualServer.Name(),
		DisplayName:         virtualServer.DisplayName(),
		RegistrationEnabled: virtualServer.EnableRegistration(),
	}, nil
}
