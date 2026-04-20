package commands

import (
	"context"
	"fmt"
	"github.com/The127/Keyline/internal/authentication/permissions"
	"github.com/The127/Keyline/internal/behaviours"
	"github.com/The127/Keyline/internal/database"
	"github.com/The127/Keyline/internal/middlewares"
	"github.com/The127/Keyline/internal/repositories"
	"github.com/The127/Keyline/utils"

	"github.com/The127/ioc"

	"github.com/google/uuid"
)

type PatchApplication struct {
	VirtualServerName      string
	ProjectSlug            string
	ApplicationId          uuid.UUID
	DisplayName            *string
	ClaimsMappingScript    *string
	AccessTokenHeaderType  *string
	DeviceFlowEnabled      *bool
	RedirectUris           *[]string
	PostLogoutRedirectUris *[]string
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

	if command.DeviceFlowEnabled != nil {
		application.SetDeviceFlowEnabled(*command.DeviceFlowEnabled)
	}

	if command.RedirectUris != nil {
		application.SetRedirectUris(*command.RedirectUris)
	}

	if command.PostLogoutRedirectUris != nil {
		application.SetPostLogoutRedirectUris(*command.PostLogoutRedirectUris)
	}

	dbContext.Applications().Update(application)

	return &PatchApplicationResponse{}, nil
}
