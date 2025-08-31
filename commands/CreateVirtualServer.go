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

const (
	AdminRoleName = "admin"

	AdminApplicationName = "admin-ui"
)

type CreateVirtualServer struct {
	Name               string
	DisplayName        string
	EnableRegistration bool
	Require2fa         bool
}

type CreateVirtualServerResponse struct {
	Id uuid.UUID
}

func HandleCreateVirtualServer(ctx context.Context, command CreateVirtualServer) (*CreateVirtualServerResponse, error) {
	scope := middlewares.GetScope(ctx)

	virtualServerRepository := ioc.GetDependency[repositories.VirtualServerRepository](scope)
	virtualServer := repositories.NewVirtualServer(command.Name, command.DisplayName)
	virtualServer.SetEnableRegistration(command.EnableRegistration)
	virtualServer.SetRequire2fa(command.Require2fa)
	err := virtualServerRepository.Insert(ctx, virtualServer)
	if err != nil {
		return nil, fmt.Errorf("inserting virtual server: %w", err)
	}

	keyService := ioc.GetDependency[services.KeyService](scope)
	_, err = keyService.Generate(command.Name)
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

func initializeDefaultApplications(ctx context.Context, virtualServer *repositories.VirtualServer) error {
	scope := middlewares.GetScope(ctx)

	applicationRepository := ioc.GetDependency[repositories.ApplicationRepository](scope)

	adminUiApplication := repositories.NewApplication(
		virtualServer.Id(),
		AdminApplicationName,
		"Admin Application",
		[]string{
			fmt.Sprintf("http://localhost:5173/mgmt/%s/auth", virtualServer.Name()),
		},
	)
	adminUiApplication.GenerateSecret()
	adminUiApplication.SetPostLogoutRedirectUris([]string{
		fmt.Sprintf("http://localhost:5173/mgmt/%s/logout", virtualServer.Name()),
	})

	err := applicationRepository.Insert(ctx, adminUiApplication)
	if err != nil {
		return fmt.Errorf("inserting application: %w", err)
	}

	return nil
}

func initializeDefaultRoles(ctx context.Context, virtualServer *repositories.VirtualServer) error {
	scope := middlewares.GetScope(ctx)

	roleRepository := ioc.GetDependency[repositories.RoleRepository](scope)

	adminRole := repositories.NewRole(
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

func initializeDefaultTemplates(ctx context.Context, virtualServer *repositories.VirtualServer) error {
	scope := middlewares.GetScope(ctx)

	fileRepository := ioc.GetDependency[repositories.FileRepository](scope)
	templateRepository := ioc.GetDependency[repositories.TemplateRepository](scope)

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
	virtualServer *repositories.VirtualServer,
	fileRepository repositories.FileRepository,
	templateRepository repositories.TemplateRepository,
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
