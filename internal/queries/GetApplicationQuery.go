package queries

import (
	"Keyline/internal/authentication/permissions"
	"Keyline/internal/behaviours"
	"Keyline/internal/database"
	"Keyline/internal/middlewares"
	"Keyline/internal/repositories"
	"context"
	"fmt"
	"time"

	"github.com/The127/ioc"

	"github.com/google/uuid"
)

type GetApplication struct {
	VirtualServerName string
	ProjectSlug       string
	ApplicationId     uuid.UUID
}

func (a GetApplication) LogRequest() bool {
	return true
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
	Id                  uuid.UUID
	Name                string
	DisplayName         string
	Type                repositories.ApplicationType
	RedirectUris        []string
	PostLogoutUris      []string
	SystemApplication   bool
	ClaimsMappingScript *string
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

func HandleGetApplication(ctx context.Context, query GetApplication) (*GetApplicationResult, error) {
	scope := middlewares.GetScope(ctx)
	dbContext := ioc.GetDependency[database.Context](scope)

	virtualServerFilter := repositories.NewVirtualServerFilter().
		Name(query.VirtualServerName)
	virtualServer, err := dbContext.VirtualServers().Single(ctx, virtualServerFilter)
	if err != nil {
		return nil, fmt.Errorf("searching virtual servers: %w", err)
	}

	projectFilter := repositories.NewProjectFilter().
		VirtualServerId(virtualServer.Id()).
		Slug(query.ProjectSlug)
	project, err := dbContext.Projects().Single(ctx, projectFilter)
	if err != nil {
		return nil, fmt.Errorf("getting project: %w", err)
	}

	applicationFilter := repositories.NewApplicationFilter().
		VirtualServerId(virtualServer.Id()).
		ProjectId(project.Id()).
		Id(query.ApplicationId)
	application, err := dbContext.Applications().FirstOrNil(ctx, applicationFilter)
	if err != nil {
		return nil, fmt.Errorf("searching application: %w", err)
	}

	if application == nil {
		return nil, nil
	}

	return &GetApplicationResult{
		Id:                  application.Id(),
		Name:                application.Name(),
		DisplayName:         application.DisplayName(),
		Type:                application.Type(),
		RedirectUris:        application.RedirectUris(),
		PostLogoutUris:      application.PostLogoutRedirectUris(),
		SystemApplication:   application.SystemApplication(),
		ClaimsMappingScript: application.ClaimsMappingScript(),
		CreatedAt:           application.AuditCreatedAt(),
		UpdatedAt:           application.AuditUpdatedAt(),
	}, nil
}
