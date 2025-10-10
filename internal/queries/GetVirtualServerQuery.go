package queries

import (
	"Keyline/internal/config"
	"Keyline/internal/middlewares"
	"Keyline/internal/repositories"
	"Keyline/ioc"
	"context"
	"time"

	"github.com/google/uuid"
)

type GetVirtualServerQuery struct {
	VirtualServerName string
}

type GetVirtualServerResponse struct {
	Id                       uuid.UUID
	Name                     string
	DisplayName              string
	RegistrationEnabled      bool
	Require2fa               bool
	RequireEmailVerification bool
	SigningAlgorithm         config.SigningAlgorithm
	CreatedAt                time.Time
	UpdatedAt                time.Time
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
		Id:                       virtualServer.Id(),
		Name:                     virtualServer.Name(),
		DisplayName:              virtualServer.DisplayName(),
		RegistrationEnabled:      virtualServer.EnableRegistration(),
		Require2fa:               virtualServer.Require2fa(),
		RequireEmailVerification: virtualServer.RequireEmailVerification(),
		SigningAlgorithm:         virtualServer.SigningAlgorithm(),
		CreatedAt:                virtualServer.AuditCreatedAt(),
		UpdatedAt:                virtualServer.AuditUpdatedAt(),
	}, nil
}
