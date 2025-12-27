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

type ListResourceServers struct {
	PagedQuery
	VirtualServerName string
	ProjectSlug       string
	SearchText        string
}

func (a ListResourceServers) LogRequest() bool {
	return true
}

func (a ListResourceServers) LogResponse() bool {
	return false
}

func (a ListResourceServers) IsAllowed(ctx context.Context) (behaviours.PolicyResult, error) {
	return behaviours.PermissionBasedPolicy(ctx, permissions.ResourceServerView)
}

func (a ListResourceServers) GetRequestName() string {
	return "ListResourceServers"
}

type ListResourceServersResponse struct {
	PagedResponse[ListResourceServersResponseItem]
}

type ListResourceServersResponseItem struct {
	Id   uuid.UUID
	Slug string
	Name string
}

func HandleListResourceServers(ctx context.Context, query ListResourceServers) (*ListResourceServersResponse, error) {
	scope := middlewares.GetScope(ctx)
	dbContext := ioc.GetDependency[database.Context](scope)

	virtualServerFilter := repositories.NewVirtualServerFilter().Name(query.VirtualServerName)
	virtualServer, err := dbContext.VirtualServers().FirstOrErr(ctx, virtualServerFilter)
	if err != nil {
		return nil, fmt.Errorf("getting virtual server: %w", err)
	}

	projectFilter := repositories.NewProjectFilter().VirtualServerId(virtualServer.Id()).Slug(query.ProjectSlug)
	project, err := dbContext.Projects().FirstOrErr(ctx, projectFilter)
	if err != nil {
		return nil, fmt.Errorf("getting project: %w", err)
	}

	resourceServerFilter := repositories.NewResourceServerFilter().
		VirtualServerId(virtualServer.Id()).
		ProjectId(project.Id())
	resourceServers, total, err := dbContext.ResourceServers().List(ctx, resourceServerFilter)
	if err != nil {
		return nil, fmt.Errorf("getting resource servers: %w", err)
	}

	items := utils.MapSlice(resourceServers, func(x *repositories.ResourceServer) ListResourceServersResponseItem {
		return ListResourceServersResponseItem{
			Id:   x.Id(),
			Slug: x.Slug(),
			Name: x.Name(),
		}
	})

	return &ListResourceServersResponse{
		PagedResponse: NewPagedResponse(items, total),
	}, nil
}
