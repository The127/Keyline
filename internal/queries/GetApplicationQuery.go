package queries

import (
	"Keyline/internal/authentication/permissions"
	"Keyline/internal/behaviours"
	"Keyline/internal/middlewares"
	"Keyline/internal/repositories"
	"Keyline/ioc"
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type GetApplication struct {
	VirtualServerName string
	ApplicationId     uuid.UUID
}

func (a GetApplication) LogResponse() bool {
	return false
}

func (a GetApplication) IsAllowed(ctx context.Context) (behaviours.PolicyResult, error) {
	return behaviours.PermissionBasedPolicy(ctx, permissions.ApplicationView)
}

func (a GetApplication) GetRequestName() string {
	return "GetApplication"
}

type GetApplicationResult struct {
	Id                uuid.UUID
	Name              string
	DisplayName       string
	Type              repositories.ApplicationType
	RedirectUris      []string
	PostLogoutUris    []string
	SystemApplication bool
	CreatedAt         time.Time
	UpdatedAt         time.Time
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
		Id:                application.Id(),
		Name:              application.Name(),
		DisplayName:       application.DisplayName(),
		Type:              application.Type(),
		RedirectUris:      application.RedirectUris(),
		PostLogoutUris:    application.PostLogoutRedirectUris(),
		SystemApplication: application.SystemApplication(),
		CreatedAt:         application.AuditCreatedAt(),
		UpdatedAt:         application.AuditUpdatedAt(),
	}, nil
}
