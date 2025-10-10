package queries

import (
	"Keyline/internal/middlewares"
	repositories2 "Keyline/internal/repositories"
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

	virtualServerRepository := ioc.GetDependency[repositories2.VirtualServerRepository](scope)
	virtualServerFilter := repositories2.NewVirtualServerFilter().Name(query.VirtualServerName)
	virtualServer, err := virtualServerRepository.Single(ctx, virtualServerFilter)
	if err != nil {
		return nil, fmt.Errorf("getting virtual server: %w", err)
	}

	roleRepository := ioc.GetDependency[repositories2.RoleRepository](scope)
	roleFilter := repositories2.NewRoleFilter().
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
