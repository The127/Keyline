package commands

import (
	"github.com/The127/Keyline/internal/authentication"
	"github.com/The127/Keyline/internal/authentication/permissions"
	"github.com/The127/Keyline/internal/behaviours"
	"github.com/The127/Keyline/internal/database"
	"github.com/The127/Keyline/internal/events"
	"github.com/The127/Keyline/internal/middlewares"
	"github.com/The127/Keyline/internal/repositories"
	"github.com/The127/Keyline/utils"
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
		Role: role,
	})
	if err != nil {
		return nil, fmt.Errorf("raising event: %w", err)
	}

	return &CreateRoleResponse{
		Id: role.Id(),
	}, nil
}
