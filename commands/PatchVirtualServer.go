package commands

import (
	"Keyline/ioc"
	"Keyline/middlewares"
	"Keyline/repositories"
	"context"
	"fmt"
)

type PatchVirtualServer struct {
	VirtualServerName string
	DisplayName       *string
}

type PatchVirtualServerResponse struct{}

func HandlePatchVirtualServer(ctx context.Context, command PatchVirtualServer) (*PatchVirtualServerResponse, error) {
	scope := middlewares.GetScope(ctx)

	virtualServerRepository := ioc.GetDependency[repositories.VirtualServerRepository](scope)
	virtualServerFilter := repositories.NewVirtualServerFilter().Name(command.VirtualServerName)
	virtualServer, err := virtualServerRepository.Single(ctx, virtualServerFilter)
	if err != nil {
		return nil, fmt.Errorf("getting virtual server: %w", err)
	}

	if command.DisplayName != nil {
		virtualServer.SetDisplayName(*command.DisplayName)
	}

	err = virtualServerRepository.Update(ctx, virtualServer)
	if err != nil {
		return nil, fmt.Errorf("updating virtual server: %w", err)
	}

	return &PatchVirtualServerResponse{}, nil
}
