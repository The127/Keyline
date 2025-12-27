package commands

import (
	"Keyline/internal/authentication/permissions"
	"Keyline/internal/behaviours"
	"Keyline/internal/database"
	"Keyline/internal/middlewares"
	"Keyline/internal/repositories"
	"Keyline/utils"
	"context"
	"fmt"

	"github.com/The127/ioc"

	"github.com/google/uuid"
)

type PatchApplication struct {
	VirtualServerName     string
	ProjectSlug           string
	ApplicationId         uuid.UUID
	DisplayName           *string
	ClaimsMappingScript   *string
	AccessTokenHeaderType *string
}

func (a PatchApplication) LogRequest() bool {
	return true
}

func (a PatchApplication) LogResponse() bool {
	return true
}

func (a PatchApplication) IsAllowed(ctx context.Context) (behaviours.PolicyResult, error) {
	return behaviours.PermissionBasedPolicy(ctx, permissions.ApplicationUpdate)
}

func (a PatchApplication) GetRequestName() string {
	return "PatchApplication"
}

type PatchApplicationResponse struct{}

func HandlePatchApplication(ctx context.Context, command PatchApplication) (*PatchApplicationResponse, error) {
	scope := middlewares.GetScope(ctx)
	dbContext := ioc.GetDependency[database.Context](scope)

	virtualServerFilter := repositories.NewVirtualServerFilter().Name(command.VirtualServerName)
	virtualServer, err := dbContext.VirtualServers().FirstOrErr(ctx, virtualServerFilter)
	if err != nil {
		return nil, fmt.Errorf("getting virtual server: %w", err)
	}

	projectFilter := repositories.NewProjectFilter().VirtualServerId(virtualServer.Id()).Slug(command.ProjectSlug)
	project, err := dbContext.Projects().FirstOrErr(ctx, projectFilter)
	if err != nil {
		return nil, fmt.Errorf("getting project: %w", err)
	}

	applicationFilter := repositories.NewApplicationFilter().
		VirtualServerId(virtualServer.Id()).
		ProjectId(project.Id()).
		Id(command.ApplicationId)
	application, err := dbContext.Applications().FirstOrErr(ctx, applicationFilter)
	if err != nil {
		return nil, fmt.Errorf("getting application: %w", err)
	}

	if command.DisplayName != nil {
		application.SetDisplayName(*command.DisplayName)
	}

	if command.ClaimsMappingScript != nil {
		if application.SystemApplication() {
			return nil, fmt.Errorf("cannot update claims mapping script for system application: %w", utils.ErrHttpBadRequest)
		}

		if *command.ClaimsMappingScript == "" {
			application.SetClaimsMappingScript(nil)
		} else {
			application.SetClaimsMappingScript(command.ClaimsMappingScript)
		}
	}

	if command.AccessTokenHeaderType != nil {
		application.SetAccessTokenHeaderType(*command.AccessTokenHeaderType)
	}

	dbContext.Applications().Update(application)
	return &PatchApplicationResponse{}, nil
}
