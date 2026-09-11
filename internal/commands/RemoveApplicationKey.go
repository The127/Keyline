package commands

import (
	"context"
	"fmt"
	"github.com/The127/Keyline/internal/authentication/permissions"
	"github.com/The127/Keyline/internal/behaviours"
	"github.com/The127/Keyline/internal/database"
	"github.com/The127/Keyline/internal/middlewares"
	"github.com/The127/Keyline/internal/repositories"

	"github.com/The127/ioc"

	"github.com/google/uuid"
)

type RemoveApplicationKey struct {
	VirtualServerName string
	ProjectSlug       string
	ApplicationId     uuid.UUID
	Kid               string
}

func (a RemoveApplicationKey) LogRequest() bool {
	return true
}

func (a RemoveApplicationKey) LogResponse() bool {
	return true
}

func (a RemoveApplicationKey) IsAllowed(ctx context.Context) (behaviours.PolicyResult, error) {
	return behaviours.PermissionBasedPolicy(ctx, permissions.ApplicationRemoveKey)
}

func (a RemoveApplicationKey) GetRequestName() string {
	return "RemoveApplicationKey"
}

type RemoveApplicationKeyResponse struct{}

func HandleRemoveApplicationKey(ctx context.Context, command RemoveApplicationKey) (*RemoveApplicationKeyResponse, error) {
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

	applicationKeyFilter := repositories.NewApplicationKeyFilter().
		ApplicationId(application.Id()).
		Kid(command.Kid)
	applicationKey, err := dbContext.ApplicationKeys().FirstOrErr(ctx, applicationKeyFilter)
	if err != nil {
		return nil, fmt.Errorf("getting application key: %w", err)
	}

	dbContext.ApplicationKeys().Delete(applicationKey.Id())

	return &RemoveApplicationKeyResponse{}, nil
}
