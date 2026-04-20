package commands

import (
	"context"
	"fmt"
	"github.com/The127/Keyline/internal/authentication/permissions"
	"github.com/The127/Keyline/internal/behaviours"
	"github.com/The127/Keyline/internal/database"
	"github.com/The127/Keyline/internal/middlewares"
	"github.com/The127/Keyline/internal/repositories"

	"github.com/The127/ioc"

	"github.com/google/uuid"
)

type CreateServiceUser struct {
	VirtualServerName string
	Username          string
}

func (a CreateServiceUser) LogRequest() bool {
	return true
}

func (a CreateServiceUser) LogResponse() bool {
	return true
}

func (a CreateServiceUser) IsAllowed(ctx context.Context) (behaviours.PolicyResult, error) {
	return behaviours.PermissionBasedPolicy(ctx, permissions.ServiceUserCreate)
}

func (a CreateServiceUser) GetRequestName() string {
	return "CreateServiceUser"
}

type CreateServiceUserResponse struct {
	Id uuid.UUID
}

func HandleCreateServiceUser(ctx context.Context, command CreateServiceUser) (*CreateServiceUserResponse, error) {
	scope := middlewares.GetScope(ctx)
	dbContext := ioc.GetDependency[database.Context](scope)

	virtualServerFilter := repositories.NewVirtualServerFilter().Name(command.VirtualServerName)
	virtualServer, err := dbContext.VirtualServers().FirstOrErr(ctx, virtualServerFilter)
	if err != nil {
		return nil, fmt.Errorf("getting virtual server: %w", err)
	}

	user := repositories.NewServiceUser(
		command.Username,
		virtualServer.Id(),
	)
	dbContext.Users().Insert(user)

	return &CreateServiceUserResponse{
		Id: user.Id(),
	}, nil
}
