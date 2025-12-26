package queries

import (
	"Keyline/internal/authentication/permissions"
	"Keyline/internal/behaviours"
	"Keyline/internal/database"
	"Keyline/internal/middlewares"
	"Keyline/internal/repositories"
	"context"
	"fmt"
	"time"

	"github.com/The127/ioc"

	"github.com/google/uuid"
)

type GetResourceServer struct {
	VirtualServerName string
	ProjectSlug       string
	ResourceServerId  uuid.UUID
}

func (a GetResourceServer) LogRequest() bool {
	return true
}

func (a GetResourceServer) LogResponse() bool {
	return false
}

func (a GetResourceServer) IsAllowed(ctx context.Context) (behaviours.PolicyResult, error) {
	return behaviours.PermissionBasedPolicy(ctx, permissions.ResourceServerView)
}

func (a GetResourceServer) GetRequestName() string {
	return "GetResourceServer"
}

type GetResourceServerResponse struct {
	Id          uuid.UUID
	Slug        string
	Name        string
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func HandleGetResourceServer(ctx context.Context, query GetResourceServer) (*GetResourceServerResponse, error) {
	scope := middlewares.GetScope(ctx)
	dbContext := ioc.GetDependency[database.Context](scope)

	virtualServerFilter := repositories.NewVirtualServerFilter().Name(query.VirtualServerName)
	virtualServer, err := dbContext.VirtualServers().Single(ctx, virtualServerFilter)
	if err != nil {
		return nil, fmt.Errorf("getting virtual server: %w", err)
	}

	projectFilter := repositories.NewProjectFilter().VirtualServerId(virtualServer.Id()).Slug(query.ProjectSlug)
	project, err := dbContext.Projects().Single(ctx, projectFilter)
	if err != nil {
		return nil, fmt.Errorf("getting project: %w", err)
	}

	resourceServerFilter := repositories.NewResourceServerFilter().
		VirtualServerId(virtualServer.Id()).
		ProjectId(project.Id()).
		Id(query.ResourceServerId)
	resourceServer, err := dbContext.ResourceServers().Single(ctx, resourceServerFilter)
	if err != nil {
		return nil, fmt.Errorf("getting resource server: %w", err)
	}

	return &GetResourceServerResponse{
		Id:          resourceServer.Id(),
		Slug:        resourceServer.Slug(),
		Name:        resourceServer.Name(),
		Description: resourceServer.Description(),
		CreatedAt:   resourceServer.AuditCreatedAt(),
		UpdatedAt:   resourceServer.AuditUpdatedAt(),
	}, nil
}
