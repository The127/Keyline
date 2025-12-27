package commands

import (
	"Keyline/internal/authentication"
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

type CreateResourceServer struct {
	VirtualServerName string
	ProjectSlug       string
	Slug              string
	Name              string
	Description       string
}

func (a CreateResourceServer) LogRequest() bool {
	return true
}

func (a CreateResourceServer) LogResponse() bool {
	return true
}

func (a CreateResourceServer) IsAllowed(ctx context.Context) (behaviours.PolicyResult, error) {
	return behaviours.PermissionBasedPolicy(ctx, permissions.ResourceServerCreate)
}

func (a CreateResourceServer) GetRequestName() string {
	return "CreateResourceServer"
}

type CreateResourceServerResponse struct {
	Id uuid.UUID
}

func HandleCreateResourceServer(ctx context.Context, command CreateResourceServer) (*CreateResourceServerResponse, error) {
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

	if project.SystemProject() {
		currentUser := authentication.GetCurrentUser(ctx)
		hasPermissionResult := currentUser.HasPermission(permissions.SystemUser)
		if !hasPermissionResult.IsSuccess() {
			return nil, fmt.Errorf("creating resource servers in system project requires system user permission: %w", utils.ErrHttpUnauthorized)
		}
	}

	resourceServer := repositories.NewResourceServer(virtualServer.Id(), project.Id(), command.Slug, command.Name, command.Description)
	dbContext.ResourceServers().Insert(resourceServer)

	return &CreateResourceServerResponse{
		Id: resourceServer.Id(),
	}, nil
}
