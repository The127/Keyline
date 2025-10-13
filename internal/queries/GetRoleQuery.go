package queries

import (
	"Keyline/internal/authentication/permissions"
	"Keyline/internal/behaviours"
	"Keyline/internal/middlewares"
	"Keyline/internal/repositories"
	"Keyline/ioc"
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type GetRoleQuery struct {
	VirtualServerName string
	RoleId            uuid.UUID
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
	RequireMfa  bool
	MaxTokenAge *time.Duration
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func HandleGetRole(ctx context.Context, query GetRoleQuery) (*GetRoleQueryResult, error) {
	scope := middlewares.GetScope(ctx)

	virtualServerRepository := ioc.GetDependency[repositories.VirtualServerRepository](scope)
	virtualServerFilter := repositories.NewVirtualServerFilter().Name(query.VirtualServerName)
	virtualServer, err := virtualServerRepository.Single(ctx, virtualServerFilter)
	if err != nil {
		return nil, fmt.Errorf("getting virtual server: %w", err)
	}

	roleRepository := ioc.GetDependency[repositories.RoleRepository](scope)
	roleFilter := repositories.NewRoleFilter().
		VirtualServerId(virtualServer.Id()).
		Id(query.RoleId)
	role, err := roleRepository.Single(ctx, roleFilter)
	if err != nil {
		return nil, fmt.Errorf("getting role: %w", err)
	}

	return &GetRoleQueryResult{
		Id:          role.Id(),
		Name:        role.Name(),
		Description: role.Description(),
		RequireMfa:  role.RequireMfa(),
		MaxTokenAge: role.MaxTokenAge(),
		CreatedAt:   role.AuditCreatedAt(),
		UpdatedAt:   role.AuditUpdatedAt(),
	}, nil
}
