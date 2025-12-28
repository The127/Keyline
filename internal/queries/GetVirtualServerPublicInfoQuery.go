package queries

import (
	"Keyline/internal/database"
	"Keyline/internal/middlewares"
	"Keyline/internal/repositories"
	"context"
	"fmt"

	"github.com/The127/ioc"
)

type GetVirtualServerPublicInfo struct {
	VirtualServerName string
}

// This query is public, so there is no policy.

type GetVirtualServerPublicInfoResponse struct {
	Name                string
	DisplayName         string
	RegistrationEnabled bool
}

func HandleGetVirtualServerPublicInfo(ctx context.Context, query GetVirtualServerPublicInfo) (*GetVirtualServerPublicInfoResponse, error) {
	scope := middlewares.GetScope(ctx)
	dbContext := ioc.GetDependency[database.Context](scope)

	virtualServerFilter := repositories.NewVirtualServerFilter().
		Name(query.VirtualServerName)
	virtualServer, err := dbContext.VirtualServers().FirstOrErr(ctx, virtualServerFilter)
	if err != nil {
		return nil, fmt.Errorf("searching virtual servers: %w", err)
	}

	return &GetVirtualServerPublicInfoResponse{
		Name:                virtualServer.Name(),
		DisplayName:         virtualServer.DisplayName(),
		RegistrationEnabled: virtualServer.EnableRegistration(),
	}, nil
}
