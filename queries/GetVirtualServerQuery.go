package queries

import (
	"Keyline/ioc"
	"Keyline/middlewares"
	"Keyline/repositories"
	"context"
	"github.com/google/uuid"
	"time"
)

type GetVirtualServerQuery struct {
	VirtualServerName string
}

type GetVirtualServerResponse struct {
	Id                  uuid.UUID
	Name                string
	DisplayName         string
	RegistrationEnabled bool
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

func HandleGetVirtualServerQuery(ctx context.Context, command GetVirtualServerQuery) (*GetVirtualServerResponse, error) {
	scope := middlewares.GetScope(ctx)

	virtualServerRepository := ioc.GetDependency[repositories.VirtualServerRepository](scope)
	virtualServerFilter := repositories.NewVirtualServerFilter().Name(command.VirtualServerName)
	virtualServer, err := virtualServerRepository.Single(ctx, virtualServerFilter)
	if err != nil {
		return nil, err
	}

	return &GetVirtualServerResponse{
		Id:                  virtualServer.Id(),
		Name:                virtualServer.Name(),
		DisplayName:         virtualServer.DisplayName(),
		RegistrationEnabled: virtualServer.EnableRegistration(),
		CreatedAt:           virtualServer.AuditCreatedAt(),
		UpdatedAt:           virtualServer.AuditUpdatedAt(),
	}, nil
}
