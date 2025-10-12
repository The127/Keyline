package queries

import (
	"Keyline/internal/authentication/permissions"
	"Keyline/internal/behaviours"
	"Keyline/internal/middlewares"
	"Keyline/internal/repositories"
	"Keyline/ioc"
	"Keyline/utils"
	"context"
	"fmt"

	"github.com/google/uuid"
)

type ListUsersInRole struct {
	PagedQuery
	OrderedQuery
	VirtualServerName string
	RoleId            uuid.UUID
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

	// virtualServerRepository := ioc.GetDependency[repositories.VirtualServerRepository](scope)
	// virtualServerFilter := repositories.NewVirtualServerFilter().Name(query.VirtualServerName)
	// virtualServer, err := virtualServerRepository.Single(ctx, virtualServerFilter)
	// if err != nil {
	// 	return nil, fmt.Errorf("getting virtual server: %w", err)
	// }

	userRoleAssignmentRepository := ioc.GetDependency[repositories.UserRoleAssignmentRepository](scope)
	userRoleAssignmentFilter := repositories.NewUserRoleAssignmentFilter().
		// TODO: Add virtual server filter
		RoleId(query.RoleId).
		IncludeUser()
	userRoleAssignments, totalCount, err := userRoleAssignmentRepository.List(ctx, userRoleAssignmentFilter)
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
