package queries

import (
	"Keyline/ioc"
	"Keyline/middlewares"
	"Keyline/repositories"
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type GetApplication struct {
	VirtualServerName string
	ApplicationId     uuid.UUID
}

type GetApplicationResult struct {
	Id             uuid.UUID
	Name           string
	DisplayName    string
	Type           repositories.ApplicationType
	RedirectUris   []string
	PostLogoutUris []string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func HandleGetApplication(ctx context.Context, query GetApplication) (*GetApplicationResult, error) {
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
		Id(query.ApplicationId)
	application, err := applicationRepository.First(ctx, applicationFilter)
	if err != nil {
		return nil, fmt.Errorf("searching application: %w", err)
	}

	if application == nil {
		return nil, nil
	}

	return &GetApplicationResult{
		Id:             application.Id(),
		Name:           application.Name(),
		DisplayName:    application.DisplayName(),
		Type:           application.Type(),
		RedirectUris:   application.RedirectUris(),
		PostLogoutUris: application.PostLogoutRedirectUris(),
		CreatedAt:      application.AuditCreatedAt(),
		UpdatedAt:      application.AuditUpdatedAt(),
	}, nil
}
