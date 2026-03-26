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

type CreateGroup struct {
	VirtualServerName string
	Name              string
	Description       string
}

func (a CreateGroup) LogRequest() bool {
	return true
}

func (a CreateGroup) LogResponse() bool {
	return true
}

func (a CreateGroup) IsAllowed(ctx context.Context) (behaviours.PolicyResult, error) {
	return behaviours.PermissionBasedPolicy(ctx, permissions.GroupCreate)
}

func (a CreateGroup) GetRequestName() string {
	return "CreateGroup"
}

type CreateGroupResponse struct {
	Id uuid.UUID
}

func HandleCreateGroup(ctx context.Context, command CreateGroup) (*CreateGroupResponse, error) {
	scope := middlewares.GetScope(ctx)
	dbContext := ioc.GetDependency[database.Context](scope)

	virtualServerFilter := repositories.NewVirtualServerFilter().Name(command.VirtualServerName)
	virtualServer, err := dbContext.VirtualServers().FirstOrErr(ctx, virtualServerFilter)
	if err != nil {
		return nil, fmt.Errorf("getting virtual server: %w", err)
	}

	group := repositories.NewGroup(
		virtualServer.Id(),
		command.Name,
		command.Description,
	)
	dbContext.Groups().Insert(group)

	return &CreateGroupResponse{
		Id: group.Id(),
	}, nil
}
