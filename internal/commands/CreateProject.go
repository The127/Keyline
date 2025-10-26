package commands

import (
	"Keyline/internal/middlewares"
	"Keyline/internal/repositories"
	"Keyline/ioc"
	"context"
	"fmt"

	"github.com/google/uuid"
)

type CreateProject struct {
	VirtualServerName string
	Slug              string
	Name              string
	Description       string
}

type CreateProjectResponse struct {
	Id uuid.UUID
}

func HandleCreateProject(ctx context.Context, command CreateProject) (*CreateProjectResponse, error) {
	scope := middlewares.GetScope(ctx)

	virtualServerRepository := ioc.GetDependency[repositories.VirtualServerRepository](scope)
	virtualServerFilter := repositories.NewVirtualServerFilter().Name(command.VirtualServerName)
	virtualServer, err := virtualServerRepository.Single(ctx, virtualServerFilter)
	if err != nil {
		return nil, fmt.Errorf("getting virtual server: %w", err)
	}

	projectRepository := ioc.GetDependency[repositories.ProjectRepository](scope)
	project := repositories.NewProject(virtualServer.Id(), command.Slug, command.Name, command.Description)
	err = projectRepository.Insert(ctx, project)
	if err != nil {
		return nil, fmt.Errorf("inserting project: %w", err)
	}

	return &CreateProjectResponse{
		Id: project.Id(),
	}, nil
}
