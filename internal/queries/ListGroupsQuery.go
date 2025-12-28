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

type ListGroups struct {
	PagedQuery
	OrderedQuery
	VirtualServerName string
	SearchText        string
}

func (a ListGroups) LogRequest() bool {
	return true
}

func (a ListGroups) LogResponse() bool {
	return false
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
	dbContext := ioc.GetDependency[database.Context](scope)

	virtualServerFilter := repositories.NewVirtualServerFilter().Name(query.VirtualServerName)
	virtualServer, err := dbContext.VirtualServers().FirstOrErr(ctx, virtualServerFilter)
	if err != nil {
		return nil, fmt.Errorf("getting virtual server: %w", err)
	}

	groupFilter := repositories.NewGroupFilter().
		VirtualServerId(virtualServer.Id()).
		Pagination(query.Page, query.PageSize).
		Order(query.OrderBy, query.OrderDir).
		Search(repositories.NewContainsSearchFilter(query.SearchText))
	groups, total, err := dbContext.Groups().List(ctx, groupFilter)
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
