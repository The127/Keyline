package commands

import (
	"Keyline/events"
	"Keyline/ioc"
	"Keyline/mediator"
	"Keyline/middlewares"
	"Keyline/repositories"
	"context"
	"fmt"
	"github.com/google/uuid"
	"time"
)

type CreateRole struct {
	VirtualServerName string
	Name              string
	Description       string
	RequireMfa        bool
	MaxTokenAge       time.Duration
}

type CreateRoleResponse struct {
	Id uuid.UUID
}

func HandleCreateRole(ctx context.Context, command CreateRole) (*CreateRoleResponse, error) {
	scope := middlewares.GetScope(ctx)

	virtualServerRepository := ioc.GetDependency[*repositories.VirtualServerRepository](scope)
	virtualServerFilter := repositories.NewVirtualServerFilter().Name(command.VirtualServerName)
	virtualServer, err := virtualServerRepository.Single(ctx, virtualServerFilter)
	if err != nil {
		return nil, fmt.Errorf("getting virtual server: %w", err)
	}

	roleRepository := ioc.GetDependency[repositories.RoleRepository](scope)
	role := repositories.NewRole(
		virtualServer.Id(),
		nil,
		command.Name,
		command.Description,
	)
	role.SetRequireMfa(command.RequireMfa)
	role.SetMaxTokenAge(&command.MaxTokenAge)
	err = roleRepository.Insert(ctx, role)
	if err != nil {
		return nil, fmt.Errorf("inserting role: %w", err)
	}

	m := ioc.GetDependency[*mediator.Mediator](scope)
	err = mediator.SendEvent(ctx, m, events.RoleCreatedEvent{
		RoleId: role.Id(),
	})
	if err != nil {
		return nil, fmt.Errorf("raising event: %w", err)
	}

	return &CreateRoleResponse{
		Id: role.Id(),
	}, nil
}
