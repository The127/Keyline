package queries

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

type ListUsersInRole struct {
	PagedQuery
	OrderedQuery
	VirtualServerName string
	ProjectSlug       string
	RoleId            uuid.UUID
}

func (a ListUsersInRole) LogRequest() bool {
	return true
}

func (a ListUsersInRole) LogResponse() bool {
	return false
}

func (a ListUsersInRole) IsAllowed(ctx context.Context) (behaviours.PolicyResult, error) {
	return behaviours.PermissionBasedPolicy(ctx, permissions.UserView)
}

func (a ListUsersInRole) GetRequestName() string {
	return "ListUsersInRole"
}

type ListUsersInRoleResponse struct {
	PagedResponse[ListUsersInRoleResponseItem]
}

type ListUsersInRoleResponseItem struct {
	Id          uuid.UUID
	Username    string
	DisplayName string
}

func HandleListUsersInRole(ctx context.Context, query ListUsersInRole) (*ListUsersInRoleResponse, error) {
	scope := middlewares.GetScope(ctx)
	dbContext := ioc.GetDependency[database.Context](scope)

	virtualServerFilter := repositories.NewVirtualServerFilter().Name(query.VirtualServerName)
	virtualServer, err := dbContext.VirtualServers().Single(ctx, virtualServerFilter)
	if err != nil {
		return nil, fmt.Errorf("getting virtual server: %w", err)
	}

	projectFilter := repositories.NewProjectFilter().VirtualServerId(virtualServer.Id()).Slug(query.ProjectSlug)
	_, err = dbContext.Projects().Single(ctx, projectFilter)
	if err != nil {
		return nil, fmt.Errorf("getting project: %w", err)
	}

	userRoleAssignmentFilter := repositories.NewUserRoleAssignmentFilter().
		// TODO: Add virtual server + project filter
		RoleId(query.RoleId).
		IncludeUser()
	userRoleAssignments, totalCount, err := dbContext.UserRoleAssignments().List(ctx, userRoleAssignmentFilter)
	if err != nil {
		return nil, fmt.Errorf("getting user role assignments: %w", err)
	}

	items := utils.MapSlice(userRoleAssignments, func(t *repositories.UserRoleAssignment) ListUsersInRoleResponseItem {
		userInfo := t.UserInfo()
		return ListUsersInRoleResponseItem{
			Id:          t.UserId(),
			Username:    userInfo.Username,
			DisplayName: userInfo.DisplayName,
		}
	})

	return &ListUsersInRoleResponse{
		PagedResponse: NewPagedResponse(items, totalCount),
	}, nil
}
