package commands

import (
	"Keyline/internal/middlewares"
	repositories2 "Keyline/internal/repositories"
	"Keyline/ioc"
	"context"
	"fmt"

	"github.com/google/uuid"
)

type PatchApplication struct {
	VirtualServerName string
	ApplicationId     uuid.UUID
	DisplayName       *string
}

type PatchApplicationResponse struct{}

func HandlePatchApplication(ctx context.Context, command PatchApplication) (*PatchApplicationResponse, error) {
	scope := middlewares.GetScope(ctx)

	virtualServerRepository := ioc.GetDependency[repositories2.VirtualServerRepository](scope)
	virtualServerFilter := repositories2.NewVirtualServerFilter().Name(command.VirtualServerName)
	virtualServer, err := virtualServerRepository.Single(ctx, virtualServerFilter)
	if err != nil {
		return nil, fmt.Errorf("getting virtual server: %w", err)
	}

	applicationRepository := ioc.GetDependency[repositories2.ApplicationRepository](scope)
	applicationFilter := repositories2.NewApplicationFilter().
		VirtualServerId(virtualServer.Id()).
		Id(command.ApplicationId)
	application, err := applicationRepository.Single(ctx, applicationFilter)
	if err != nil {
		return nil, fmt.Errorf("getting application: %w", err)
	}

	if command.DisplayName != nil {
		application.SetDisplayName(*command.DisplayName)
	}

	err = applicationRepository.Update(ctx, application)
	if err != nil {
		return nil, fmt.Errorf("updating application: %w", err)
	}

	return &PatchApplicationResponse{}, nil
}
