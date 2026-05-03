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

type AssignRoleToUser struct {
	VirtualServerName string
	ProjectSlug       string
	UserId            uuid.UUID
	RoleId            uuid.UUID
}

func (a AssignRoleToUser) LogRequest() bool {
	return true
}

func (a AssignRoleToUser) LogResponse() bool {
	return true
}

func (a AssignRoleToUser) IsAllowed(ctx context.Context) (behaviours.PolicyResult, error) {
	return behaviours.PermissionBasedPolicy(ctx, permissions.RoleAssign)
}

func (a AssignRoleToUser) GetRequestName() string {
	return "AssignRoleToUser"
}

type AssignRoleToUserResponse struct{}

func HandleAssignRoleToUser(ctx context.Context, command AssignRoleToUser) (*AssignRoleToUserResponse, error) {
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

	// Role assignments inside the system project drive the JWT
	// `system:<roleName>` claim that AuthenticationMiddleware honours
	// directly via roles.AllRoles. Granting one is the same trust
	// decision as creating, renaming, or deleting one (see
	// CreateRole / PatchRole / DeleteRole) and requires the same
	// gate. Without it, any caller with RoleAssign (i.e. every VS
	// admin) could assign the system-project `system-admin` role to
	// themselves on the initial VS and silently promote to
	// SystemAdmin permissions on next login.
	if project.SystemProject() {
		currentUser := authentication.GetCurrentUser(ctx)
		if !currentUser.HasPermission(permissions.SystemUser).IsSuccess() {
			return nil, fmt.Errorf("assigning roles in system project requires system user permission: %w", utils.ErrHttpUnauthorized)
		}
	}

	roleFilter := repositories.NewRoleFilter().
		Id(command.RoleId).
		VirtualServerId(virtualServer.Id()).
		ProjectId(project.Id())

	_, err = dbContext.Roles().FirstOrErr(ctx, roleFilter)
	if err != nil {
		return nil, fmt.Errorf("getting role: %w", err)
	}

	_, err = dbContext.Users().FirstOrErr(ctx, repositories.NewUserFilter().Id(command.UserId).VirtualServerId(virtualServer.Id()))
	if err != nil {
		return nil, fmt.Errorf("getting user: %w", err)
	}

	userRoleAssignment := repositories.NewUserRoleAssignment(command.UserId, command.RoleId, nil)
	dbContext.UserRoleAssignments().Insert(userRoleAssignment)

	return &AssignRoleToUserResponse{}, nil
}
