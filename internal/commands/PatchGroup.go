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

type PatchGroup struct {
	VirtualServerName string
	GroupId           uuid.UUID
	Name              *string
	Description       *string
}

func (a PatchGroup) LogRequest() bool {
	return true
}

func (a PatchGroup) LogResponse() bool {
	return true
}

func (a PatchGroup) IsAllowed(ctx context.Context) (behaviours.PolicyResult, error) {
	return behaviours.PermissionBasedPolicy(ctx, permissions.GroupUpdate)
}

func (a PatchGroup) GetRequestName() string {
	return "PatchGroup"
}

type PatchGroupResponse struct{}

func HandlePatchGroup(ctx context.Context, command PatchGroup) (*PatchGroupResponse, error) {
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
	group, err := dbContext.Groups().FirstOrErr(ctx, groupFilter)
	if err != nil {
		return nil, fmt.Errorf("getting group: %w", err)
	}

	if command.Name != nil {
		group.SetName(*command.Name)
	}
	if command.Description != nil {
		group.SetDescription(*command.Description)
	}

	dbContext.Groups().Update(group)
	return &PatchGroupResponse{}, nil
}
