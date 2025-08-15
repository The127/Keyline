package commands

import (
	"Keyline/ioc"
	"Keyline/middlewares"
	"Keyline/repositories"
	"Keyline/services"
	"Keyline/templates"
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

	err = initializeDefaultTemplates(ctx, virtualServer)
	if err != nil {
		return nil, fmt.Errorf("initializing default templates: %w", err)
	}

	return &CreateVirtualServerResponse{
		Id: virtualServer.Id(),
	}, nil
}

func initializeDefaultTemplates(ctx context.Context, virtualServer *repositories.VirtualServer) error {
	scope := middlewares.GetScope(ctx)

	fileRepository := ioc.GetDependency[*repositories.FileRepository](scope)
	templateRepository := ioc.GetDependency[*repositories.TemplateRepository](scope)

	err2 := insertTemplate(
		ctx,
		"email_verification_template",
		virtualServer,
		fileRepository,
		templateRepository)
	if err2 != nil {
		return err2
	}

	return nil
}

func insertTemplate(
	ctx context.Context,
	templateName string,
	virtualServer *repositories.VirtualServer,
	fileRepository *repositories.FileRepository,
	templateRepository *repositories.TemplateRepository,
) error {
	file := repositories.NewFile(templateName, "text/plain", templates.DefaultEmailVerificationTemplate)
	err := fileRepository.Insert(ctx, file)
	if err != nil {
		return fmt.Errorf("inserting %s file: %w", templateName, err)
	}

	t := repositories.NewTemplate(virtualServer.Id(), file.Id(), repositories.EmailVerificationMailTemplate)
	err = templateRepository.Insert(ctx, t)
	if err != nil {
		return fmt.Errorf("inserting %s template: %w", templateName, err)
	}

	return nil
}
