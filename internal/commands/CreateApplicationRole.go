package commands

import (
	"Keyline/internal/authentication/permissions"
	"Keyline/internal/behaviours"
	"Keyline/internal/events"
	"Keyline/internal/middlewares"
	"Keyline/internal/repositories"
	"Keyline/ioc"
	"Keyline/mediator"
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type CreateApplicationRole struct {
	VirtualServerName string
	ApplicationId     uuid.UUID
	Name              string
	Description       string
	RequireMfa        bool
	MaxTokenAge       time.Duration
}

func (a CreateApplicationRole) LogRequest() bool {
	return true
}

func (a CreateApplicationRole) LogResponse() bool {
	return true
}

func (a CreateApplicationRole) IsAllowed(ctx context.Context) (behaviours.PolicyResult, error) {
	return behaviours.PermissionBasedPolicy(ctx, permissions.RoleCreate) // TODO: this is maybe the wrong permission
}

func (a CreateApplicationRole) GetRequestName() string {
	return "CreateApplicationRole"
}

type CreateApplicationRoleResponse struct {
	Id uuid.UUID
}

func HandleCreateApplicationRole(ctx context.Context, command CreateApplicationRole) (*CreateApplicationRoleResponse, error) {
	scope := middlewares.GetScope(ctx)

	virtualServerRepository := ioc.GetDependency[repositories.VirtualServerRepository](scope)
	virtualServerFilter := repositories.NewVirtualServerFilter().Name(command.VirtualServerName)
	virtualServer, err := virtualServerRepository.Single(ctx, virtualServerFilter)
	if err != nil {
		return nil, fmt.Errorf("getting virtual server: %w", err)
	}

	applicationRepository := ioc.GetDependency[repositories.ApplicationRepository](scope)
	applicationFilter := repositories.NewApplicationFilter().
		VirtualServerId(virtualServer.Id()).
		Id(command.ApplicationId)
	application, err := applicationRepository.Single(ctx, applicationFilter)
	if err != nil {
		return nil, fmt.Errorf("getting application: %w", err)
	}

	roleRepository := ioc.GetDependency[repositories.RoleRepository](scope)
	role := repositories.NewApplicationRole(
		virtualServer.Id(),
		application.Id(),
		command.Name,
		command.Description,
	)
	role.SetRequireMfa(command.RequireMfa)
	role.SetMaxTokenAge(&command.MaxTokenAge)
	err = roleRepository.Insert(ctx, role)
	if err != nil {
		return nil, fmt.Errorf("inserting role: %w", err)
	}

	m := ioc.GetDependency[mediator.Mediator](scope)
	err = mediator.SendEvent(ctx, m, events.RoleCreatedEvent{
		RoleId: role.Id(),
	})
	if err != nil {
		return nil, fmt.Errorf("raising event: %w", err)
	}

	return &CreateApplicationRoleResponse{
		Id: role.Id(),
	}, nil
}
