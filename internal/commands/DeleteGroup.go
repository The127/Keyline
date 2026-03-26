package commands

import (
	"Keyline/internal/authentication/permissions"
	"Keyline/internal/behaviours"
	"Keyline/internal/database"
	"Keyline/internal/middlewares"
	"Keyline/internal/repositories"
	"context"
	"fmt"

	"github.com/The127/ioc"

	"github.com/google/uuid"
)

type DeleteGroup struct {
	VirtualServerName string
	GroupId           uuid.UUID
}

func (a DeleteGroup) LogRequest() bool {
	return true
}

func (a DeleteGroup) LogResponse() bool {
	return true
}

func (a DeleteGroup) IsAllowed(ctx context.Context) (behaviours.PolicyResult, error) {
	return behaviours.PermissionBasedPolicy(ctx, permissions.GroupDelete)
}

func (a DeleteGroup) GetRequestName() string {
	return "DeleteGroup"
}

type DeleteGroupResponse struct{}

func HandleDeleteGroup(ctx context.Context, command DeleteGroup) (*DeleteGroupResponse, error) {
	scope := middlewares.GetScope(ctx)
	dbContext := ioc.GetDependency[database.Context](scope)

	virtualServerFilter := repositories.NewVirtualServerFilter().Name(command.VirtualServerName)
	virtualServer, err := dbContext.VirtualServers().FirstOrErr(ctx, virtualServerFilter)
	if err != nil {
		return nil, fmt.Errorf("getting virtual server: %w", err)
	}

	groupFilter := repositories.NewGroupFilter().
		VirtualServerId(virtualServer.Id()).
		Id(command.GroupId)
	group, err := dbContext.Groups().FirstOrNil(ctx, groupFilter)
	if err != nil {
		return nil, fmt.Errorf("getting group: %w", err)
	}

	if group == nil {
		return &DeleteGroupResponse{}, nil
	}

	dbContext.Groups().Delete(group.Id())

	return &DeleteGroupResponse{}, nil
}
