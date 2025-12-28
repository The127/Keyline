package commands

import (
	"Keyline/internal/authentication/permissions"
	"Keyline/internal/behaviours"
	"Keyline/internal/config"
	"Keyline/internal/database"
	"Keyline/internal/middlewares"
	"Keyline/internal/repositories"
	"Keyline/internal/services"
	"Keyline/templates"
	"Keyline/utils"
	"context"
	"fmt"
	"strings"

	"github.com/The127/go-clock"

	"github.com/The127/ioc"
	"github.com/google/uuid"
)

const (
	AdminRoleName       = "admin"
	SystemAdminRoleName = "system-admin"

	AdminApplicationName = "admin-ui"
)

type CreateVirtualServerAdmin struct {
	Username     string
	DisplayName  string
	PrimaryEmail string
	PasswordHash string
	Roles        []string
}

type CreateVirtualServerServiceUser struct {
	Username  string
	Roles     []string
	PublicKey struct {
		Pem string
		Kid string
	}
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

	CreateSystemAdminRole bool

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
	dbContext := ioc.GetDependency[database.Context](scope)

	roleIdsByFullyQualifiedName := make(map[string]uuid.UUID)

	virtualServer := repositories.NewVirtualServer(command.Name, command.DisplayName)
	virtualServer.SetEnableRegistration(command.EnableRegistration)
	virtualServer.SetRequire2fa(command.Require2fa)
	virtualServer.SetSigningAlgorithm(command.SigningAlgorithm)

	dbContext.VirtualServers().Insert(virtualServer)

	clockService := ioc.GetDependency[clock.Service](scope)

	keyService := ioc.GetDependency[services.KeyService](scope)
	_, err := keyService.Generate(clockService, command.Name, command.SigningAlgorithm)
	if err != nil {
		return nil, fmt.Errorf("generating keypair: %w", err)
	}
	initializeDefaultTemplates(ctx, virtualServer)

	systemProject := repositories.NewSystemProject(virtualServer.Id())
	dbContext.Projects().Insert(systemProject)

	initDefaultAppsResult := initializeDefaultApplications(ctx, virtualServer, systemProject)
	defaultRolesResult := initializeDefaultAdminRoles(ctx, virtualServer, systemProject, command.CreateSystemAdminRole)

	roleIdsByFullyQualifiedName[fmt.Sprintf("%s:%s", systemProject.Slug(), AdminRoleName)] = defaultRolesResult.adminRoleId
	if defaultRolesResult.systemAdminRoleId != nil {
		roleIdsByFullyQualifiedName[fmt.Sprintf("%s:%s", systemProject.Slug(), SystemAdminRoleName)] = *defaultRolesResult.systemAdminRoleId
	}

	for _, project := range command.Projects {
		newProject := repositories.NewProject(virtualServer.Id(), project.Slug, project.Name, project.Description)
		dbContext.Projects().Insert(newProject)

		for _, app := range project.Applications {
			newApp := repositories.NewApplication(
				virtualServer.Id(),
				newProject.Id(),
				app.Name,
				app.DisplayName,
				repositories.ApplicationType(app.Type),
				app.RedirectUris,
			)
			if app.HashedSecret != nil {
				newApp.SetHashedSecret(*app.HashedSecret)
			}
			newApp.SetPostLogoutRedirectUris(app.PostLogoutUris)
			dbContext.Applications().Insert(newApp)
		}

		for _, role := range project.Roles {
			newRole := repositories.NewRole(virtualServer.Id(), newProject.Id(), role.Name, role.Description)
			dbContext.Roles().Insert(newRole)
			roleIdsByFullyQualifiedName[fmt.Sprintf("%s:%s", project.Slug, role.Name)] = newRole.Id()
		}

		for _, resourceServer := range project.ResourceServers {
			newResourceServer := repositories.NewResourceServer(virtualServer.Id(), newProject.Id(), resourceServer.Slug, resourceServer.Name, resourceServer.Description)
			dbContext.ResourceServers().Insert(newResourceServer)
		}
	}

	if command.Admin != nil {
		initialAdminUser := repositories.NewUser(command.Admin.Username, command.Admin.DisplayName, command.Admin.PrimaryEmail, virtualServer.Id())
		initialAdminUser.SetEmailVerified(true)
		dbContext.Users().Insert(initialAdminUser)

		initialAdminCredential := repositories.NewCredential(initialAdminUser.Id(), &repositories.CredentialPasswordDetails{
			HashedPassword: command.Admin.PasswordHash,
			Temporary:      false,
		})
		dbContext.Credentials().Insert(initialAdminCredential)

		adminRoleAssignment := repositories.NewUserRoleAssignment(
			initialAdminUser.Id(),
			defaultRolesResult.adminRoleId,
			nil,
		)
		dbContext.UserRoleAssignments().Insert(adminRoleAssignment)

		if command.CreateSystemAdminRole {
			systemAdminRoleAssignment := repositories.NewUserRoleAssignment(
				initialAdminUser.Id(),
				*defaultRolesResult.systemAdminRoleId,
				nil,
			)
			dbContext.UserRoleAssignments().Insert(systemAdminRoleAssignment)
		}

		err = assignRoles(ctx, initialAdminUser.Id(), command.Admin.Roles, roleIdsByFullyQualifiedName)
		if err != nil {
			return nil, fmt.Errorf("assigning roles to admin user: %w", err)
		}
	}

	for _, serviceUser := range command.ServiceUsers {
		newServiceUser := repositories.NewServiceUser(serviceUser.Username, virtualServer.Id())
		dbContext.Users().Insert(newServiceUser)

		associatedPublicKey := repositories.NewCredential(
			newServiceUser.Id(),
			&repositories.CredentialServiceUserKey{
				Kid:       serviceUser.PublicKey.Kid,
				PublicKey: serviceUser.PublicKey.Pem,
			},
		)
		dbContext.Credentials().Insert(associatedPublicKey)

		err = assignRoles(ctx, newServiceUser.Id(), serviceUser.Roles, roleIdsByFullyQualifiedName)
		if err != nil {
			return nil, fmt.Errorf("assigning roles to service user: %w", err)
		}
	}

	return &CreateVirtualServerResponse{
		Id:                   virtualServer.Id(),
		SystemProjectId:      systemProject.Id(),
		SystemProjectSlug:    systemProject.Slug(),
		AdminUiApplicationId: initDefaultAppsResult.adminUidApplicationId,
		AdminRoleId:          defaultRolesResult.adminRoleId,
	}, nil
}

func assignRoles(ctx context.Context, userId uuid.UUID, roleList []string, roleIdsByFullyQualifiedName map[string]uuid.UUID) error {
	scope := middlewares.GetScope(ctx)
	dbContext := ioc.GetDependency[database.Context](scope)

	for _, configuredRole := range roleList {
		if !strings.Contains(configuredRole, ":") {
			return fmt.Errorf("role %s does not contain project slug", configuredRole)
		}

		roleId, ok := roleIdsByFullyQualifiedName[configuredRole]
		if !ok {
			return fmt.Errorf("role %s not found", configuredRole)
		}

		newRoleAssignment := repositories.NewUserRoleAssignment(userId, roleId, nil)
		dbContext.UserRoleAssignments().Insert(newRoleAssignment)
	}

	return nil
}

type createDefaultApplicationResult struct {
	adminUidApplicationId uuid.UUID
}

func initializeDefaultApplications(ctx context.Context, virtualServer *repositories.VirtualServer, systemProject *repositories.Project) *createDefaultApplicationResult {
	scope := middlewares.GetScope(ctx)
	dbContext := ioc.GetDependency[database.Context](scope)

	adminUiApplication := repositories.NewApplication(virtualServer.Id(), systemProject.Id(), AdminApplicationName, "Admin Application", repositories.ApplicationTypePublic, []string{
		fmt.Sprintf("%s/mgmt/%s/auth", config.C.Frontend.ExternalUrl, virtualServer.Name()),
	})
	adminUiApplication.GenerateSecret()
	adminUiApplication.SetPostLogoutRedirectUris([]string{
		fmt.Sprintf("%s/mgmt/%s/logout", config.C.Frontend.ExternalUrl, virtualServer.Name()),
	})
	adminUiApplication.SetSystemApplication(true)

	dbContext.Applications().Insert(adminUiApplication)

	return &createDefaultApplicationResult{
		adminUidApplicationId: adminUiApplication.Id(),
	}
}

type createDefaultAdminUiRolesResult struct {
	adminRoleId       uuid.UUID
	systemAdminRoleId *uuid.UUID
}

func initializeDefaultAdminRoles(ctx context.Context, virtualServer *repositories.VirtualServer, project *repositories.Project, createSystemAdminRole bool) *createDefaultAdminUiRolesResult {
	scope := middlewares.GetScope(ctx)
	dbContext := ioc.GetDependency[database.Context](scope)

	adminRole := repositories.NewRole(
		virtualServer.Id(),
		project.Id(),
		AdminRoleName,
		"Administrator role",
	)
	dbContext.Roles().Insert(adminRole)

	var systemAdminRoleId *uuid.UUID
	if createSystemAdminRole {
		systemAdminRole := repositories.NewRole(
			virtualServer.Id(),
			project.Id(),
			SystemAdminRoleName,
			"System administrator role",
		)
		dbContext.Roles().Insert(systemAdminRole)

		systemAdminRoleId = utils.Ptr(systemAdminRole.Id())
	}

	return &createDefaultAdminUiRolesResult{
		adminRoleId:       adminRole.Id(),
		systemAdminRoleId: systemAdminRoleId,
	}
}

func initializeDefaultTemplates(ctx context.Context, virtualServer *repositories.VirtualServer) {
	insertTemplate(
		ctx,
		"email_verification_template",
		virtualServer,
	)
}

func insertTemplate(
	ctx context.Context,
	templateName string,
	virtualServer *repositories.VirtualServer,
) {
	scope := middlewares.GetScope(ctx)
	dbContext := ioc.GetDependency[database.Context](scope)

	file := repositories.NewFile(templateName, "text/plain", templates.DefaultEmailVerificationTemplate)
	dbContext.Files().Insert(file)

	t := repositories.NewTemplate(virtualServer.Id(), file.Id(), repositories.EmailVerificationMailTemplate)
	dbContext.Templates().Insert(t)
}
