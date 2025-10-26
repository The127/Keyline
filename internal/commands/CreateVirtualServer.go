package commands

import (
	"Keyline/internal/authentication/permissions"
	"Keyline/internal/behaviours"
	"Keyline/internal/clock"
	"Keyline/internal/config"
	"Keyline/internal/middlewares"
	"Keyline/internal/repositories"
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

func (a CreateVirtualServer) LogRequest() bool {
	return true
}

func (a CreateVirtualServer) LogResponse() bool {
	return true
}

func (a CreateVirtualServer) IsAllowed(ctx context.Context) (behaviours.PolicyResult, error) {
	return behaviours.PermissionBasedPolicy(ctx, permissions.VirtualServerCreate)
}

func (a CreateVirtualServer) GetRequestName() string {
	return "CreateVirtualServer"
}

type CreateVirtualServerResponse struct {
	Id                   uuid.UUID
	AdminUiApplicationId uuid.UUID
	AdminRoleId          uuid.UUID
}

func HandleCreateVirtualServer(ctx context.Context, command CreateVirtualServer) (*CreateVirtualServerResponse, error) {
	scope := middlewares.GetScope(ctx)

	virtualServerRepository := ioc.GetDependency[repositories.VirtualServerRepository](scope)

	virtualServer := repositories.NewVirtualServer(command.Name, command.DisplayName)
	virtualServer.SetEnableRegistration(command.EnableRegistration)
	virtualServer.SetRequire2fa(command.Require2fa)
	virtualServer.SetSigningAlgorithm(command.SigningAlgorithm)

	err := virtualServerRepository.Insert(ctx, virtualServer)
	if err != nil {
		return nil, fmt.Errorf("inserting virtual server: %w", err)
	}

	clockService := ioc.GetDependency[clock.Service](scope)

	keyService := ioc.GetDependency[services.KeyService](scope)
	_, err = keyService.Generate(clockService, command.Name, command.SigningAlgorithm)
	if err != nil {
		return nil, fmt.Errorf("generating keypair: %w", err)
	}
	err = initializeDefaultTemplates(ctx, virtualServer)
	if err != nil {
		return nil, fmt.Errorf("initializing default templates: %w", err)
	}

	initDefaultAppsResult, err := initializeDefaultApplications(ctx, virtualServer)
	if err != nil {
		return nil, fmt.Errorf("initializing default applications: %w", err)
	}

	return &CreateVirtualServerResponse{
		Id:                   virtualServer.Id(),
		AdminUiApplicationId: initDefaultAppsResult.adminUidApplicationId,
		AdminRoleId:          initDefaultAppsResult.adminRoleId,
	}, nil
}

type createDefaultApplicationResult struct {
	adminUidApplicationId uuid.UUID
	adminRoleId           uuid.UUID
}

func initializeDefaultApplications(ctx context.Context, virtualServer *repositories.VirtualServer) (*createDefaultApplicationResult, error) {
	scope := middlewares.GetScope(ctx)

	projectRepository := ioc.GetDependency[repositories.ProjectRepository](scope)

	systemProject := repositories.NewProject(virtualServer.Id(), "system", "Keyline Internal", "Internal project for keyline management")
	err := projectRepository.Insert(ctx, systemProject)
	if err != nil {
		return nil, fmt.Errorf("inserting project: %w", err)
	}

	applicationRepository := ioc.GetDependency[repositories.ApplicationRepository](scope)

	adminUiApplication := repositories.NewApplication(virtualServer.Id(), systemProject.Id(), AdminApplicationName, "Admin Application", repositories.ApplicationTypePublic, []string{
		fmt.Sprintf("%s/mgmt/%s/auth", config.C.Frontend.ExternalUrl, virtualServer.Name()),
	})
	adminUiApplication.GenerateSecret()
	adminUiApplication.SetPostLogoutRedirectUris([]string{
		fmt.Sprintf("%s/mgmt/%s/logout", config.C.Frontend.ExternalUrl, virtualServer.Name()),
	})
	adminUiApplication.SetSystemApplication(true)

	err = applicationRepository.Insert(ctx, adminUiApplication)
	if err != nil {
		return nil, fmt.Errorf("inserting application: %w", err)
	}

	createAdminUidRolesResult, err := initializeDefaultAdminUiRoles(ctx, virtualServer, adminUiApplication)
	if err != nil {
		return nil, fmt.Errorf("initializing default roles: %w", err)
	}

	return &createDefaultApplicationResult{
		adminUidApplicationId: adminUiApplication.Id(),
		adminRoleId:           createAdminUidRolesResult.adminRoleId,
	}, nil
}

type createDefaultAdminUiRolesResult struct {
	adminRoleId uuid.UUID
}

func initializeDefaultAdminUiRoles(ctx context.Context, virtualServer *repositories.VirtualServer, application *repositories.Application) (*createDefaultAdminUiRolesResult, error) {
	scope := middlewares.GetScope(ctx)

	roleRepository := ioc.GetDependency[repositories.RoleRepository](scope)

	adminRole := repositories.NewApplicationRole(
		virtualServer.Id(),
		application.Id(),
		AdminRoleName,
		"Administrator role",
	)
	adminRole.SetRequireMfa(true)
	adminRole.SetMaxTokenAge(nil)

	err := roleRepository.Insert(ctx, adminRole)
	if err != nil {
		return nil, fmt.Errorf("inserting role: %w", err)
	}

	return &createDefaultAdminUiRolesResult{
		adminRoleId: adminRole.Id(),
	}, nil
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
