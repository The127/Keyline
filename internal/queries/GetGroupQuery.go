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

type GetGroupQuery struct {
	VirtualServerName string
	GroupId           uuid.UUID
}

func (a GetGroupQuery) LogRequest() bool {
	return true
}

func (a GetGroupQuery) LogResponse() bool {
	return false
}

func (a GetGroupQuery) IsAllowed(ctx context.Context) (behaviours.PolicyResult, error) {
	return behaviours.PermissionBasedPolicy(ctx, permissions.GroupView)
}

func (a GetGroupQuery) GetRequestName() string {
	return "GetGroupQuery"
}

type GetGroupQueryResult struct {
	Id          uuid.UUID
	Name        string
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func HandleGetGroup(ctx context.Context, query GetGroupQuery) (*GetGroupQueryResult, error) {
	scope := middlewares.GetScope(ctx)
	dbContext := ioc.GetDependency[database.Context](scope)

	virtualServerFilter := repositories.NewVirtualServerFilter().Name(query.VirtualServerName)
	virtualServer, err := dbContext.VirtualServers().FirstOrErr(ctx, virtualServerFilter)
	if err != nil {
		return nil, fmt.Errorf("getting virtual server: %w", err)
	}

	groupFilter := repositories.NewGroupFilter().
		VirtualServerId(virtualServer.Id()).
		Id(query.GroupId)
	group, err := dbContext.Groups().FirstOrErr(ctx, groupFilter)
	if err != nil {
		return nil, fmt.Errorf("getting group: %w", err)
	}

	return &GetGroupQueryResult{
		Id:          group.Id(),
		Name:        group.Name(),
		Description: group.Description(),
		CreatedAt:   group.AuditCreatedAt(),
		UpdatedAt:   group.AuditUpdatedAt(),
	}, nil
}
