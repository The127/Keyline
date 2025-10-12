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

type ListGroups struct {
	PagedQuery
	OrderedQuery
	VirtualServerName string
	SearchText        string
}

func (a ListGroups) IsAllowed(ctx context.Context) (behaviours.PolicyResult, error) {
	return behaviours.PermissionBasedPolicy(ctx, permissions.GroupView)
}

func (a ListGroups) GetRequestName() string {
	return "ListGroups"
}

type ListGroupsResponse struct {
	PagedResponse[ListGroupsResponseItem]
}

type ListGroupsResponseItem struct {
	Id          uuid.UUID
	Name        string
	Description string
}

func HandleListGroups(ctx context.Context, query ListGroups) (*ListGroupsResponse, error) {
	scope := middlewares.GetScope(ctx)

	virtualServerRepository := ioc.GetDependency[repositories.VirtualServerRepository](scope)
	virtualServerFilter := repositories.NewVirtualServerFilter().Name(query.VirtualServerName)
	virtualServer, err := virtualServerRepository.Single(ctx, virtualServerFilter)
	if err != nil {
		return nil, fmt.Errorf("getting virtual server: %w", err)
	}

	groupRepository := ioc.GetDependency[repositories.GroupRepository](scope)
	groupFilter := repositories.NewGroupFilter().
		VirtualServerId(virtualServer.Id()).
		Pagination(query.Page, query.PageSize).
		Order(query.OrderBy, query.OrderDir).
		Search(repositories.NewContainsSearchFilter(query.SearchText))
	groups, total, err := groupRepository.List(ctx, groupFilter)
	if err != nil {
		return nil, fmt.Errorf("searching groups: %w", err)
	}

	items := utils.MapSlice(groups, func(t *repositories.Group) ListGroupsResponseItem {
		return ListGroupsResponseItem{
			Id:          t.Id(),
			Name:        t.Name(),
			Description: t.Description(),
		}
	})

	return &ListGroupsResponse{
		PagedResponse: NewPagedResponse(items, total),
	}, nil
}
