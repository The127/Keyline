package queries

import (
	"Keyline/internal/middlewares"
	"Keyline/internal/repositories"
	"Keyline/ioc"
	"Keyline/utils"
	"context"
	"fmt"

	"github.com/google/uuid"
)

type ListRoles struct {
	PagedQuery
	OrderedQuery
	VirtualServerName string
	ApplicationId     *uuid.UUID
	SearchText        string
}

type ListRolesResponse struct {
	PagedResponse[ListRolesResponseItem]
}

type ListRolesResponseItem struct {
	Id          uuid.UUID
	Name        string
	DisplayName string
}

func HandleListRoles(ctx context.Context, query ListRoles) (*ListRolesResponse, error) {
	scope := middlewares.GetScope(ctx)

	virtualServerRepository := ioc.GetDependency[repositories.VirtualServerRepository](scope)
	virtualServerFilter := repositories.NewVirtualServerFilter().
		Name(query.VirtualServerName)
	virtualServer, err := virtualServerRepository.Single(ctx, virtualServerFilter)
	if err != nil {
		return nil, fmt.Errorf("searching virtual servers: %w", err)
	}

	roleRepository := ioc.GetDependency[repositories.RoleRepository](scope)
	roleFilter := repositories.NewRoleFilter().
		VirtualServerId(virtualServer.Id()).
		Pagination(query.Page, query.PageSize).
		Order(query.OrderBy, query.OrderDir).
		Search(query.SearchText)

	if query.ApplicationId != nil {
		roleFilter = roleFilter.ApplicationId(*query.ApplicationId)
	}

	roles, total, err := roleRepository.List(ctx, roleFilter)
	if err != nil {
		return nil, fmt.Errorf("searching roles: %w", err)
	}

	items := utils.MapSlice(roles, func(t *repositories.Role) ListRolesResponseItem {
		return ListRolesResponseItem{
			Id:   t.Id(),
			Name: t.Name(),
		}
	})

	return &ListRolesResponse{
		PagedResponse: NewPagedResponse(items, total),
	}, nil
}
