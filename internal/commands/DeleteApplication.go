package commands

import (
	"Keyline/internal/authentication/permissions"
	"Keyline/internal/behaviours"
	db "Keyline/internal/database"
	"Keyline/internal/middlewares"
	"Keyline/internal/repositories"
	"Keyline/utils"
	"context"
	"fmt"

	"github.com/The127/ioc"

	"github.com/google/uuid"
)

type DeleteApplication struct {
	VirtualServerName string
	ProjectSlug       string
	ApplicationId     uuid.UUID
}

func (a DeleteApplication) LogRequest() bool {
	return true
}

func (a DeleteApplication) LogResponse() bool {
	return true
}

func (a DeleteApplication) IsAllowed(ctx context.Context) (behaviours.PolicyResult, error) {
	return behaviours.PermissionBasedPolicy(ctx, permissions.ApplicationDelete)
}

func (a DeleteApplication) GetRequestName() string {
	return "DeleteApplication"
}

type DeleteApplicationResponse struct{}

func HandleDeleteApplication(ctx context.Context, command DeleteApplication) (*DeleteApplicationResponse, error) {
	scope := middlewares.GetScope(ctx)
	dbContext := ioc.GetDependency[db.Context](scope)

	virtualServerFilter := repositories.NewVirtualServerFilter().Name(command.VirtualServerName)
	virtualServer, err := dbContext.VirtualServers().Single(ctx, virtualServerFilter)
	if err != nil {
		return nil, fmt.Errorf("getting virtual server: %w", err)
	}

	projectFilter := repositories.NewProjectFilter().VirtualServerId(virtualServer.Id()).Slug(command.ProjectSlug)
	project, err := dbContext.Projects().Single(ctx, projectFilter)
	if err != nil {
		return nil, fmt.Errorf("getting project: %w", err)
	}

	applicationFilter := repositories.NewApplicationFilter().
		VirtualServerId(virtualServer.Id()).
		ProjectId(project.Id()).
		Id(command.ApplicationId)
	application, err := dbContext.Applications().FirstOrNil(ctx, applicationFilter)
	if err != nil {
		return nil, fmt.Errorf("getting application: %w", err)
	}

	if application == nil {
		return &DeleteApplicationResponse{}, nil
	}

	if application.SystemApplication() {
		return nil, fmt.Errorf("cannot delete system application: %w", utils.ErrHttpBadRequest)
	}

	dbContext.Applications().Delete(application.Id())

	return &DeleteApplicationResponse{}, nil
}
