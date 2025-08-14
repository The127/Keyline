package commands

import (
	"Keyline/ioc"
	"Keyline/middlewares"
	"Keyline/repositories"
	"Keyline/services"
	"context"
	"fmt"
	"github.com/google/uuid"
)

type CreateVirtualServer struct {
	Name               string
	DisplayName        string
	EnableRegistration bool
}

type CreateVirtualServerResponse struct {
	Id uuid.UUID
}

func HandleCreateVirtualServer(ctx context.Context, command CreateVirtualServer) (*CreateVirtualServerResponse, error) {
	scope := middlewares.GetScope(ctx)

	virtualServerRepository := ioc.GetDependency[*repositories.VirtualServerRepository](scope)
	virtualServer := repositories.NewVirtualServer(command.Name, command.DisplayName).
		SetEnableRegistration(command.EnableRegistration)
	err := virtualServerRepository.Insert(ctx, virtualServer)
	if err != nil {
		return nil, fmt.Errorf("inserting virtual server: %w", err)
	}

	keyService := ioc.GetDependency[services.KeyService](scope)
	_, err = keyService.Generate(command.Name)
	if err != nil {
		return nil, fmt.Errorf("generating keypair: %w", err)
	}

	return &CreateVirtualServerResponse{
		Id: virtualServer.Id(),
	}, nil
}
