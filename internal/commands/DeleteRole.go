package commands

import (
	"context"
	"fmt"
	"github.com/The127/Keyline/internal/authentication"
	"github.com/The127/Keyline/internal/authentication/permissions"
	"github.com/The127/Keyline/internal/behaviours"
	"github.com/The127/Keyline/internal/database"
	"github.com/The127/Keyline/internal/middlewares"
	"github.com/The127/Keyline/internal/repositories"
	"github.com/The127/Keyline/utils"

	"github.com/The127/ioc"

	"github.com/google/uuid"
)

type DeleteRole struct {
	VirtualServerName string
	ProjectSlug       string
	RoleId            uuid.UUID
}

func (a DeleteRole) LogRequest() bool {
	return true
}

func (a DeleteRole) LogResponse() bool {
	return true
}

func (a DeleteRole) IsAllowed(ctx context.Context) (behaviours.PolicyResult, error) {
	return behaviours.PermissionBasedPolicy(ctx, permissions.RoleDelete)
}

func (a DeleteRole) GetRequestName() string {
	return "DeleteRole"
}

type DeleteRoleResponse struct{}

func HandleDeleteRole(ctx context.Context, command DeleteRole) (*DeleteRoleResponse, error) {
	scope := middlewares.GetScope(ctx)
	dbContext := ioc.GetDependency[database.Context](scope)

	virtualServerFilter := repositories.NewVirtualServerFilter().Name(command.VirtualServerName)
	virtualServer, err := dbContext.VirtualServers().FirstOrErr(ctx, virtualServerFilter)
	if err != nil {
		return nil, fmt.Errorf("getting virtual server: %w", err)
	}

	projectFilter := repositories.NewProjectFilter().
		VirtualServerId(virtualServer.Id()).
		Slug(command.ProjectSlug)
	project, err := dbContext.Projects().FirstOrErr(ctx, projectFilter)
	if err != nil {
		return nil, fmt.Errorf("getting project: %w", err)
	}

	// Mirror of CreateRole's system-project gate. Deleting the system-
	// project `admin` or `system-admin` role would either brick the VS
	// or strip a SaaS operator of permissions; either way it's a
	// privileged operation that must require the same SystemUser perm
	// CreateRole demands for inserts.
	if project.SystemProject() {
		currentUser := authentication.GetCurrentUser(ctx)
		if !currentUser.HasPermission(permissions.SystemUser).IsSuccess() {
			return nil, fmt.Errorf("deleting roles in system project requires system user permission: %w", utils.ErrHttpUnauthorized)
		}
	}

	roleFilter := repositories.NewRoleFilter().
		VirtualServerId(virtualServer.Id()).
		ProjectId(project.Id()).
		Id(command.RoleId)
	role, err := dbContext.Roles().FirstOrNil(ctx, roleFilter)
	if err != nil {
		return nil, fmt.Errorf("getting role: %w", err)
	}

	if role == nil {
		return &DeleteRoleResponse{}, nil
	}

	dbContext.Roles().Delete(role.Id())

	return &DeleteRoleResponse{}, nil
}
