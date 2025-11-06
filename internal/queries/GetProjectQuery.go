package queries

import (
	"Keyline/internal/authentication/permissions"
	"Keyline/internal/behaviours"
	"Keyline/internal/middlewares"
	"Keyline/internal/repositories"
	"context"
	"fmt"
	"time"

	"github.com/The127/ioc"

	"github.com/google/uuid"
)

type GetProject struct {
	VirtualServerName string
	ProjectSlug       string
}

func (a GetProject) LogRequest() bool {
	return true
}

func (a GetProject) LogResponse() bool {
	return false
}

func (a GetProject) IsAllowed(ctx context.Context) (behaviours.PolicyResult, error) {
	return behaviours.PermissionBasedPolicy(ctx, permissions.ProjectView)
}

func (a GetProject) GetRequestName() string {
	return "GetProject"
}

type GetProjectResponse struct {
	Id            uuid.UUID
	Slug          string
	Name          string
	Description   string
	SystemProject bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func HandleGetProject(ctx context.Context, query GetProject) (*GetProjectResponse, error) {
	scope := middlewares.GetScope(ctx)

	virtualServerRepository := ioc.GetDependency[repositories.VirtualServerRepository](scope)
	virtualServerFilter := repositories.NewVirtualServerFilter().Name(query.VirtualServerName)
	virtualServer, err := virtualServerRepository.Single(ctx, virtualServerFilter)
	if err != nil {
		return nil, fmt.Errorf("getting virtual server: %w", err)
	}

	projectRepository := ioc.GetDependency[repositories.ProjectRepository](scope)
	projectFilter := repositories.NewProjectFilter().
		VirtualServerId(virtualServer.Id()).
		Slug(query.ProjectSlug)
	project, err := projectRepository.Single(ctx, projectFilter)
	if err != nil {
		return nil, fmt.Errorf("getting project: %w", err)
	}

	return &GetProjectResponse{
		Id:            project.Id(),
		Slug:          project.Slug(),
		Name:          project.Name(),
		Description:   project.Description(),
		SystemProject: project.SystemProject(),
		CreatedAt:     project.AuditCreatedAt(),
		UpdatedAt:     project.AuditUpdatedAt(),
	}, nil
}
