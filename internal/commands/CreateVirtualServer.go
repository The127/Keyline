package commands

import (
	"Keyline/internal/authentication/permissions"
	"Keyline/internal/behaviours"
	"Keyline/internal/clock"
	"Keyline/internal/config"
	"Keyline/internal/logging"
	"Keyline/internal/middlewares"
	"Keyline/internal/repositories"
	"Keyline/internal/services"
	"Keyline/templates"
	"context"
	"fmt"
	"strings"

	"github.com/The127/ioc"
	"github.com/The127/mediatr"

	"github.com/google/uuid"
)

const (
	AdminRoleName = "admin"

	AdminApplicationName = "admin-ui"
)

type CreateVirtualServerAdmin struct {
	Username     string
	DisplayName  string
	PrimaryEmail string
	PasswordHash string
}

type CreateVirtualServerServiceUser struct {
	Username  string
	Roles     []string
	PublicKey string
}

type CreateVirtualServerProjectResourceServer struct {
	Slug        string
	Name        string
	Description string
}

type CreateVirtualServerProjectRole struct {
	Name        string
	Description string
}

type CreateVirtualServerProjectApplication struct {
	Name           string
	DisplayName    string
	Type           string
	HashedSecret   *string
	RedirectUris   []string
	PostLogoutUris []string
}

type CreateVirtualServerProject struct {
	Slug        string
	Name        string
	Description string

	Applications    []CreateVirtualServerProjectApplication
	Roles           []CreateVirtualServerProjectRole
	ResourceServers []CreateVirtualServerProjectResourceServer
}

type CreateVirtualServer struct {
	Name               string
	DisplayName        string
	EnableRegistration bool
	SigningAlgorithm   config.SigningAlgorithm
	Require2fa         bool

	Admin        *CreateVirtualServerAdmin
	ServiceUsers []CreateVirtualServerServiceUser
	Projects     []CreateVirtualServerProject
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
	SystemProjectId      uuid.UUID
	SystemProjectSlug    string
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

	projectRepository := ioc.GetDependency[repositories.ProjectRepository](scope)

	systemProject := repositories.NewProject(virtualServer.Id(), "system", "Keyline Internal", "Internal project for keyline management")
	err = projectRepository.Insert(ctx, systemProject)
	if err != nil {
		return nil, fmt.Errorf("inserting project: %w", err)
	}

	initDefaultAppsResult, err := initializeDefaultApplications(ctx, virtualServer, systemProject)
	if err != nil {
		return nil, fmt.Errorf("initializing default applications: %w", err)
	}

	m := ioc.GetDependency[mediatr.Mediator](scope)
	for _, project := range command.Projects {
		_, err = mediatr.Send[*CreateProjectResponse](ctx, m, CreateProject{
			VirtualServerName: virtualServer.Name(),
			Slug:              project.Slug,
			Name:              project.Name,
			Description:       project.Description,
		})
		if err != nil {
			return nil, fmt.Errorf("creating project: %w", err)
		}

		for _, app := range project.Applications {
			_, err = mediatr.Send[*CreateApplicationResponse](ctx, m, CreateApplication{
				VirtualServerName:      virtualServer.Name(),
				ProjectSlug:            project.Slug,
				Name:                   app.Name,
				DisplayName:            app.DisplayName,
				Type:                   repositories.ApplicationType(app.Type),
				RedirectUris:           app.RedirectUris,
				PostLogoutRedirectUris: app.PostLogoutUris,
				HashedSecret:           app.HashedSecret,
			})
			if err != nil {
				return nil, fmt.Errorf("creating application: %w", err)
			}
		}

		for _, role := range project.Roles {
			_, err = mediatr.Send[*CreateRoleResponse](ctx, m, CreateRole{
				VirtualServerName: virtualServer.Name(),
				ProjectSlug:       project.Slug,
				Name:              role.Name,
				Description:       role.Description,
			})
			if err != nil {
				return nil, fmt.Errorf("creating role: %w", err)
			}
		}

		for _, resourceServer := range project.ResourceServers {
			_, err = mediatr.Send[*CreateResourceServerResponse](ctx, m, CreateResourceServer{
				VirtualServerName: virtualServer.Name(),
				ProjectSlug:       project.Slug,
				Slug:              resourceServer.Slug,
				Name:              resourceServer.Name,
				Description:       resourceServer.Description,
			})
			if err != nil {
				return nil, fmt.Errorf("creating resource server: %w", err)
			}
		}
	}

	if command.Admin != nil {
		initialAdminUserInfo, err := mediatr.Send[*CreateUserResponse](ctx, m, CreateUser{
			VirtualServerName: virtualServer.Name(),
			DisplayName:       command.Admin.DisplayName,
			Username:          command.Admin.Username,
			Email:             command.Admin.PrimaryEmail,
			EmailVerified:     true,
		})
		if err != nil {
			return nil, fmt.Errorf("creating admin user: %w", err)
		}

		credentialRepository := ioc.GetDependency[repositories.CredentialRepository](scope)
		initialAdminCredential := repositories.NewCredential(initialAdminUserInfo.Id, &repositories.CredentialPasswordDetails{
			HashedPassword: command.Admin.PasswordHash,
			Temporary:      false,
		})
		err = credentialRepository.Insert(ctx, initialAdminCredential)
		if err != nil {
			logging.Logger.Fatalf("failed to create initial admin credential: %v", err)
		}

		_, err = mediatr.Send[*AssignRoleToUserResponse](ctx, m, AssignRoleToUser{
			VirtualServerName: virtualServer.Name(),
			ProjectSlug:       systemProject.Slug(),
			UserId:            initialAdminUserInfo.Id,
			RoleId:            initDefaultAppsResult.adminRoleId,
		})
		if err != nil {
			return nil, fmt.Errorf("assigning admin role to admin user: %w", err)
		}
	}

	for _, serviceUser := range command.ServiceUsers {
		serviceUserResponse, err := mediatr.Send[*CreateServiceUserResponse](ctx, m, CreateServiceUser{
			VirtualServerName: virtualServer.Name(),
			Username:          serviceUser.Username,
		})
		if err != nil {
			return nil, fmt.Errorf("creating service user: %w", err)
		}

		_, err = mediatr.Send[*AssociateServiceUserPublicKeyResponse](ctx, m, AssociateServiceUserPublicKey{
			VirtualServerName: virtualServer.Name(),
			ServiceUserId:     serviceUserResponse.Id,
			PublicKey:         serviceUser.PublicKey,
		})
		if err != nil {
			return nil, fmt.Errorf("associating service user public key: %w", err)
		}

		for _, configuredRole := range serviceUser.Roles {
			if strings.Contains(configuredRole, " ") {
				split := strings.Split(configuredRole, " ")
				projectSlug := split[0]
				roleName := split[1]

				projectRepository := ioc.GetDependency[repositories.ProjectRepository](scope)
				projectFilter := repositories.NewProjectFilter().VirtualServerId(virtualServer.Id()).Slug(projectSlug)
				project, err := projectRepository.Single(ctx, projectFilter)
				if err != nil {
					logging.Logger.Fatalf("failed to get project: %v", err)
				}

				roleRepository := ioc.GetDependency[repositories.RoleRepository](scope)
				roleFilter := repositories.NewRoleFilter().
					VirtualServerId(virtualServer.Id()).
					ProjectId(project.Id()).
					Name(roleName)
				role, err := roleRepository.Single(ctx, roleFilter)
				if err != nil {
					logging.Logger.Fatalf("failed to get role: %v", err)
				}

				_, err = mediatr.Send[*AssignRoleToUserResponse](ctx, m, AssignRoleToUser{
					VirtualServerName: virtualServer.Name(),
					ProjectSlug:       projectSlug,
					UserId:            serviceUserResponse.Id,
					RoleId:            role.Id(),
				})
				if err != nil {
					logging.Logger.Fatalf("failed to assign role to service user: %v", err)
				}
			} else {
				roleRepository := ioc.GetDependency[repositories.RoleRepository](scope)
				roleFilter := repositories.NewRoleFilter().
					VirtualServerId(virtualServer.Id()).
					Name(configuredRole)
				role, err := roleRepository.Single(ctx, roleFilter)
				if err != nil {
					logging.Logger.Fatalf("failed to get role: %v", err)
				}

				_, err = mediatr.Send[*AssignRoleToUserResponse](ctx, m, AssignRoleToUser{
					VirtualServerName: virtualServer.Name(),
					UserId:            serviceUserResponse.Id,
					RoleId:            role.Id(),
				})
				if err != nil {
					logging.Logger.Fatalf("failed to assign role to service user: %v", err)
				}
			}
		}
	}

	return &CreateVirtualServerResponse{
		Id:                   virtualServer.Id(),
		SystemProjectId:      systemProject.Id(),
		SystemProjectSlug:    systemProject.Slug(),
		AdminUiApplicationId: initDefaultAppsResult.adminUidApplicationId,
		AdminRoleId:          initDefaultAppsResult.adminRoleId,
	}, nil
}

type createDefaultApplicationResult struct {
	adminUidApplicationId uuid.UUID
	adminRoleId           uuid.UUID
}

func initializeDefaultApplications(ctx context.Context, virtualServer *repositories.VirtualServer, systemProject *repositories.Project) (*createDefaultApplicationResult, error) {
	scope := middlewares.GetScope(ctx)

	applicationRepository := ioc.GetDependency[repositories.ApplicationRepository](scope)

	adminUiApplication := repositories.NewApplication(virtualServer.Id(), systemProject.Id(), AdminApplicationName, "Admin Application", repositories.ApplicationTypePublic, []string{
		fmt.Sprintf("%s/mgmt/%s/auth", config.C.Frontend.ExternalUrl, virtualServer.Name()),
	})
	adminUiApplication.GenerateSecret()
	adminUiApplication.SetPostLogoutRedirectUris([]string{
		fmt.Sprintf("%s/mgmt/%s/logout", config.C.Frontend.ExternalUrl, virtualServer.Name()),
	})
	adminUiApplication.SetSystemApplication(true)

	err := applicationRepository.Insert(ctx, adminUiApplication)
	if err != nil {
		return nil, fmt.Errorf("inserting application: %w", err)
	}

	createAdminUidRolesResult, err := initializeDefaultAdminUiRoles(ctx, virtualServer, systemProject)
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

func initializeDefaultAdminUiRoles(ctx context.Context, virtualServer *repositories.VirtualServer, project *repositories.Project) (*createDefaultAdminUiRolesResult, error) {
	scope := middlewares.GetScope(ctx)

	roleRepository := ioc.GetDependency[repositories.RoleRepository](scope)

	adminRole := repositories.NewRole(
		virtualServer.Id(),
		project.Id(),
		AdminRoleName,
		"Administrator role",
	)

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
