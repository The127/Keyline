package commands

import (
	"Keyline/internal/authentication/permissions"
	"Keyline/internal/behaviours"
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

	virtualServerRepository := ioc.GetDependency[repositories.VirtualServerRepository](scope)
	virtualServerFilter := repositories.NewVirtualServerFilter().Name(command.VirtualServerName)
	virtualServer, err := virtualServerRepository.Single(ctx, virtualServerFilter)
	if err != nil {
		return nil, fmt.Errorf("getting virtual server: %w", err)
	}

	projectRepository := ioc.GetDependency[repositories.ProjectRepository](scope)
	projectFilter := repositories.NewProjectFilter().VirtualServerId(virtualServer.Id()).Slug(command.ProjectSlug)
	project, err := projectRepository.Single(ctx, projectFilter)
	if err != nil {
		return nil, fmt.Errorf("getting project: %w", err)
	}

	applicationRepository := ioc.GetDependency[repositories.ApplicationRepository](scope)
	applicationFilter := repositories.NewApplicationFilter().
		VirtualServerId(virtualServer.Id()).
		ProjectId(project.Id()).
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
