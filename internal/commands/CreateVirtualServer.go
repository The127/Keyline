package commands

import (
	"Keyline/internal/config"
	"Keyline/internal/middlewares"
	repositories2 "Keyline/internal/repositories"
	"Keyline/internal/services"
	"Keyline/ioc"
	"Keyline/templates"
	"context"
	"fmt"

	"github.com/google/uuid"
)

const (
	AdminRoleName = "admin"

	AdminApplicationName = "admin-ui"
)

type CreateVirtualServer struct {
	Name               string
	DisplayName        string
	EnableRegistration bool
	Require2fa         bool
	SigningAlgorithm   config.SigningAlgorithm
}

type CreateVirtualServerResponse struct {
	Id uuid.UUID
}

func HandleCreateVirtualServer(ctx context.Context, command CreateVirtualServer) (*CreateVirtualServerResponse, error) {
	scope := middlewares.GetScope(ctx)

	virtualServerRepository := ioc.GetDependency[repositories2.VirtualServerRepository](scope)

	virtualServer := repositories2.NewVirtualServer(command.Name, command.DisplayName)
	virtualServer.SetEnableRegistration(command.EnableRegistration)
	virtualServer.SetRequire2fa(command.Require2fa)
	virtualServer.SetSigningAlgorithm(command.SigningAlgorithm)

	err := virtualServerRepository.Insert(ctx, virtualServer)
	if err != nil {
		return nil, fmt.Errorf("inserting virtual server: %w", err)
	}

	keyService := ioc.GetDependency[services.KeyService](scope)
	_, err = keyService.Generate(command.Name, command.SigningAlgorithm)
	if err != nil {
		return nil, fmt.Errorf("generating keypair: %w", err)
	}

	err = initializeDefaultRoles(ctx, virtualServer)
	if err != nil {
		return nil, fmt.Errorf("initializing default roles: %w", err)
	}

	err = initializeDefaultTemplates(ctx, virtualServer)
	if err != nil {
		return nil, fmt.Errorf("initializing default templates: %w", err)
	}

	err = initializeDefaultApplications(ctx, virtualServer)
	if err != nil {
		return nil, fmt.Errorf("initializing default applications: %w", err)
	}

	return &CreateVirtualServerResponse{
		Id: virtualServer.Id(),
	}, nil
}

func initializeDefaultApplications(ctx context.Context, virtualServer *repositories2.VirtualServer) error {
	scope := middlewares.GetScope(ctx)

	applicationRepository := ioc.GetDependency[repositories2.ApplicationRepository](scope)

	adminUiApplication := repositories2.NewApplication(
		virtualServer.Id(),
		AdminApplicationName,
		"Admin Application",
		repositories2.ApplicationTypePublic,
		[]string{
			fmt.Sprintf("%s/mgmt/%s/auth", config.C.Frontend.ExternalUrl, virtualServer.Name()),
		},
	)
	adminUiApplication.GenerateSecret()
	adminUiApplication.SetPostLogoutRedirectUris([]string{
		fmt.Sprintf("%s/mgmt/%s/logout", config.C.Frontend.ExternalUrl, virtualServer.Name()),
	})
	adminUiApplication.SetSystemApplication(true)

	err := applicationRepository.Insert(ctx, adminUiApplication)
	if err != nil {
		return fmt.Errorf("inserting application: %w", err)
	}

	return nil
}

func initializeDefaultRoles(ctx context.Context, virtualServer *repositories2.VirtualServer) error {
	scope := middlewares.GetScope(ctx)

	roleRepository := ioc.GetDependency[repositories2.RoleRepository](scope)

	adminRole := repositories2.NewRole(
		virtualServer.Id(),
		nil,
		AdminRoleName,
		"Administrator role",
	)
	adminRole.SetRequireMfa(true)
	adminRole.SetMaxTokenAge(nil)

	err := roleRepository.Insert(ctx, adminRole)
	if err != nil {
		return fmt.Errorf("inserting role: %w", err)
	}

	return nil
}

func initializeDefaultTemplates(ctx context.Context, virtualServer *repositories2.VirtualServer) error {
	scope := middlewares.GetScope(ctx)

	fileRepository := ioc.GetDependency[repositories2.FileRepository](scope)
	templateRepository := ioc.GetDependency[repositories2.TemplateRepository](scope)

	err := insertTemplate(
		ctx,
		"email_verification_template",
		virtualServer,
		fileRepository,
		templateRepository)
	if err != nil {
		return err
	}

	return nil
}

func insertTemplate(
	ctx context.Context,
	templateName string,
	virtualServer *repositories2.VirtualServer,
	fileRepository repositories2.FileRepository,
	templateRepository repositories2.TemplateRepository,
) error {
	file := repositories2.NewFile(templateName, "text/plain", templates.DefaultEmailVerificationTemplate)
	err := fileRepository.Insert(ctx, file)
	if err != nil {
		return fmt.Errorf("inserting %s file: %w", templateName, err)
	}

	t := repositories2.NewTemplate(virtualServer.Id(), file.Id(), repositories2.EmailVerificationMailTemplate)
	err = templateRepository.Insert(ctx, t)
	if err != nil {
		return fmt.Errorf("inserting %s template: %w", templateName, err)
	}

	return nil
}
