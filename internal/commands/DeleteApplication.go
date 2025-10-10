package commands

import (
	"Keyline/internal/middlewares"
	repositories2 "Keyline/internal/repositories"
	"Keyline/ioc"
	"Keyline/utils"
	"context"
	"fmt"

	"github.com/google/uuid"
)

type DeleteApplication struct {
	VirtualServerName string
	ApplicationId     uuid.UUID
}

type DeleteApplicationResponse struct{}

func HandleDeleteApplication(ctx context.Context, command DeleteApplication) (*DeleteApplicationResponse, error) {
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
	application, err := applicationRepository.First(ctx, applicationFilter)
	if err != nil {
		return nil, fmt.Errorf("getting application: %w", err)
	}

	if application == nil {
		return &DeleteApplicationResponse{}, nil
	}

	if application.SystemApplication() {
		return nil, fmt.Errorf("cannot delete system application: %w", utils.ErrHttpBadRequest)
	}

	err = applicationRepository.Delete(ctx, application.Id())
	if err != nil {
		return nil, fmt.Errorf("deleting application: %w", err)
	}

	return &DeleteApplicationResponse{}, nil
}
