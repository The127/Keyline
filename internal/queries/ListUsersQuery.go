package queries

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

type ListUsers struct {
	PagedQuery
	OrderedQuery
	VirtualServerName string
	SearchText        string
}

func (a ListUsers) LogRequest() bool {
	return true
}

func (a ListUsers) LogResponse() bool {
	return false
}

func (a ListUsers) IsAllowed(ctx context.Context) (behaviours.PolicyResult, error) {
	return behaviours.PermissionBasedPolicy(ctx, permissions.UserView)
}

func (a ListUsers) GetRequestName() string {
	return "ListUsers"
}

type ListUsersResponse struct {
	PagedResponse[ListUsersResponseItem]
}

type ListUsersResponseItem struct {
	Id            uuid.UUID
	Username      string
	DisplayName   string
	Email         string
	IsServiceUser bool
}

func HandleListUsers(ctx context.Context, query ListUsers) (*ListUsersResponse, error) {
	scope := middlewares.GetScope(ctx)

	virtualServerRepository := ioc.GetDependency[repositories.VirtualServerRepository](scope)
	virtualServerFilter := repositories.NewVirtualServerFilter().
		Name(query.VirtualServerName)
	virtualServer, err := virtualServerRepository.Single(ctx, virtualServerFilter)
	if err != nil {
		return nil, fmt.Errorf("searching virtual servers: %w", err)
	}

	userRepository := ioc.GetDependency[repositories.UserRepository](scope)
	userFilter := repositories.NewUserFilter().
		VirtualServerId(virtualServer.Id()).
		Pagination(query.Page, query.PageSize).
		Order(query.OrderBy, query.OrderDir).
		Search(repositories.NewContainsSearchFilter(query.SearchText))
	users, total, err := userRepository.List(ctx, userFilter)
	if err != nil {
		return nil, fmt.Errorf("searching users: %w", err)
	}

	items := utils.MapSlice(users, func(t *repositories.User) ListUsersResponseItem {
		return ListUsersResponseItem{
			Id:            t.Id(),
			Username:      t.Username(),
			DisplayName:   t.DisplayName(),
			Email:         t.PrimaryEmail(),
			IsServiceUser: t.IsServiceUser(),
		}
	})

	return &ListUsersResponse{
		PagedResponse: NewPagedResponse(items, total),
	}, nil
}
