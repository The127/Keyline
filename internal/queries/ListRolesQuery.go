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

type ListRoles struct {
	PagedQuery
	OrderedQuery
	VirtualServerName string
	ProjectSlug       string
	SearchText        string
}

func (a ListRoles) LogRequest() bool {
	return true
}

func (a ListRoles) LogResponse() bool {
	return false
}

func (a ListRoles) IsAllowed(ctx context.Context) (behaviours.PolicyResult, error) {
	return behaviours.PermissionBasedPolicy(ctx, permissions.RoleView)
}

func (a ListRoles) GetRequestName() string {
	return "ListRoles"
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
	dbContext := ioc.GetDependency[database.Context](scope)

	virtualServerFilter := repositories.NewVirtualServerFilter().
		Name(query.VirtualServerName)
	virtualServer, err := dbContext.VirtualServers().FirstOrErr(ctx, virtualServerFilter)
	if err != nil {
		return nil, fmt.Errorf("searching virtual servers: %w", err)
	}

	projectFilter := repositories.NewProjectFilter().
		VirtualServerId(virtualServer.Id()).
		Slug(query.ProjectSlug)
	project, err := dbContext.Projects().FirstOrErr(ctx, projectFilter)
	if err != nil {
		return nil, fmt.Errorf("getting project: %w", err)
	}

	roleFilter := repositories.NewRoleFilter().
		VirtualServerId(virtualServer.Id()).
		ProjectId(project.Id()).
		Pagination(query.Page, query.PageSize).
		Order(query.OrderBy, query.OrderDir).
		Search(repositories.NewContainsSearchFilter(query.SearchText))
	roles, total, err := dbContext.Roles().List(ctx, roleFilter)
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
