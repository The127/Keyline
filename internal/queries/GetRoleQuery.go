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

type GetRoleQuery struct {
	VirtualServerName string
	ProjectSlug       string
	RoleId            uuid.UUID
}

func (a GetRoleQuery) LogRequest() bool {
	return true
}

func (a GetRoleQuery) LogResponse() bool {
	return false
}

func (a GetRoleQuery) IsAllowed(ctx context.Context) (behaviours.PolicyResult, error) {
	return behaviours.PermissionBasedPolicy(ctx, permissions.RoleView)
}

func (a GetRoleQuery) GetRequestName() string {
	return "GetRoleQuery"
}

type GetRoleQueryResult struct {
	Id          uuid.UUID
	Name        string
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func HandleGetRole(ctx context.Context, query GetRoleQuery) (*GetRoleQueryResult, error) {
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

	roleFilter := repositories.NewRoleFilter().
		VirtualServerId(virtualServer.Id()).
		ProjectId(project.Id()).
		Id(query.RoleId)
	role, err := dbContext.Roles().Single(ctx, roleFilter)
	if err != nil {
		return nil, fmt.Errorf("getting role: %w", err)
	}

	return &GetRoleQueryResult{
		Id:          role.Id(),
		Name:        role.Name(),
		Description: role.Description(),
		CreatedAt:   role.AuditCreatedAt(),
		UpdatedAt:   role.AuditUpdatedAt(),
	}, nil
}
