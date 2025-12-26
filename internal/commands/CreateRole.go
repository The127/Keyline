package commands

import (
	"Keyline/internal/authentication"
	"Keyline/internal/authentication/permissions"
	"Keyline/internal/behaviours"
	"Keyline/internal/database"
	"Keyline/internal/events"
	"Keyline/internal/middlewares"
	"Keyline/internal/repositories"
	"Keyline/utils"
	"context"
	"fmt"

	"github.com/The127/ioc"

	"github.com/The127/mediatr"

	"github.com/google/uuid"
)

type CreateRole struct {
	VirtualServerName string
	ProjectSlug       string
	Name              string
	Description       string
}

func (a CreateRole) LogRequest() bool {
	return true
}

func (a CreateRole) LogResponse() bool {
	return true
}

func (a CreateRole) IsAllowed(ctx context.Context) (behaviours.PolicyResult, error) {
	return behaviours.PermissionBasedPolicy(ctx, permissions.RoleCreate)
}

func (a CreateRole) GetRequestName() string {
	return "CreateRole"
}

type CreateRoleResponse struct {
	Id uuid.UUID
}

func HandleCreateRole(ctx context.Context, command CreateRole) (*CreateRoleResponse, error) {
	scope := middlewares.GetScope(ctx)
	dbContext := ioc.GetDependency[database.Context](scope)

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

	if project.SystemProject() {
		currentUser := authentication.GetCurrentUser(ctx)
		hasPermissionResult := currentUser.HasPermission(permissions.SystemUser)
		if !hasPermissionResult.IsSuccess() {
			return nil, fmt.Errorf("creating roles in system project requires system user permission: %w", utils.ErrHttpUnauthorized)
		}
	}

	role := repositories.NewRole(
		virtualServer.Id(),
		project.Id(),
		command.Name,
		command.Description,
	)
	dbContext.Roles().Insert(role)

	m := ioc.GetDependency[mediatr.Mediator](scope)
	err = mediatr.SendEvent(ctx, m, events.RoleCreatedEvent{
		RoleId: role.Id(),
	})
	if err != nil {
		return nil, fmt.Errorf("raising event: %w", err)
	}

	return &CreateRoleResponse{
		Id: role.Id(),
	}, nil
}
