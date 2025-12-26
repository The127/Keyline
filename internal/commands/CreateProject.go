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

type CreateProject struct {
	VirtualServerName string
	Slug              string
	Name              string
	Description       string
}

func (a CreateProject) LogRequest() bool {
	return true
}

func (a CreateProject) LogResponse() bool {
	return true
}

func (a CreateProject) IsAllowed(ctx context.Context) (behaviours.PolicyResult, error) {
	return behaviours.PermissionBasedPolicy(ctx, permissions.ProjectCreate)
}

func (a CreateProject) GetRequestName() string {
	return "CreateProject"
}

type CreateProjectResponse struct {
	Id uuid.UUID
}

func HandleCreateProject(ctx context.Context, command CreateProject) (*CreateProjectResponse, error) {
	scope := middlewares.GetScope(ctx)
	dbContext := ioc.GetDependency[database.Context](scope)

	virtualServerFilter := repositories.NewVirtualServerFilter().Name(command.VirtualServerName)
	virtualServer, err := dbContext.VirtualServers().Single(ctx, virtualServerFilter)
	if err != nil {
		return nil, fmt.Errorf("getting virtual server: %w", err)
	}

	project := repositories.NewProject(virtualServer.Id(), command.Slug, command.Name, command.Description)
	dbContext.Projects().Insert(project)

	return &CreateProjectResponse{
		Id: project.Id(),
	}, nil
}
