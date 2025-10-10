// @title       Keyline API
// @description Open source OIDC/IDP server.
// @BasePath    /

// Security schemes for the "Authorize" button (Swagger 2.0):
// @securityDefinitions.basic  BasicAuth
// @securityDefinitions.apikey BearerAuth
// @in                         header
// @name                       Authorization
package main

import (
	"Keyline/internal/behaviours"
	commands2 "Keyline/internal/commands"
	"Keyline/internal/config"
	database2 "Keyline/internal/database"
	"Keyline/internal/events"
	"Keyline/internal/jobs"
	"Keyline/internal/metrics"
	middlewares2 "Keyline/internal/middlewares"
	queries2 "Keyline/internal/queries"
	repositories2 "Keyline/internal/repositories"
	"Keyline/internal/server"
	services2 "Keyline/internal/services"
	"Keyline/ioc"
	"Keyline/logging"
	"Keyline/mediator"
	"Keyline/utils"
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"syscall"

	docs "Keyline/docs"

	"github.com/huandu/go-sqlbuilder"
)

func main() {
	config.Init()
	configureSwaggerFromConfig()

	sqlbuilder.DefaultFlavor = sqlbuilder.PostgreSQL

	logging.Init()
	metrics.Init()
	database2.Migrate()

	dc := ioc.NewDependencyCollection()
	ioc.RegisterSingleton(dc, func(dp *ioc.DependencyProvider) *sql.DB {
		return database2.ConnectToDatabase()
	})

	ioc.RegisterScoped(dc, func(dp *ioc.DependencyProvider) database2.DbService {
		return database2.NewDbService(dp)
	})
	ioc.RegisterCloseHandler(dc, func(dbService database2.DbService) error {
		return dbService.Close()
	})

	ioc.RegisterSingleton(dc, func(dp *ioc.DependencyProvider) services2.KeyCache {
		return services2.NewMemoryCache[string, services2.KeyPair]()
	})
	ioc.RegisterSingleton(dc, func(dp *ioc.DependencyProvider) services2.KeyStore {
		switch config.C.KeyStore.Mode {
		case config.KeyStoreModeDirectory:
			return services2.NewDirectoryKeyStore()

		case config.KeyStoreModeOpenBao:
			panic("not implemented yet")

		default:
			panic("not implemented")
		}
	})
	ioc.RegisterSingleton(dc, func(dp *ioc.DependencyProvider) services2.KeyService {
		return services2.NewKeyService(
			ioc.GetDependency[services2.KeyCache](dp),
			ioc.GetDependency[services2.KeyStore](dp),
		)
	})
	ioc.RegisterSingleton(dc, func(dp *ioc.DependencyProvider) services2.MailService {
		return services2.NewMailService()
	})
	ioc.RegisterSingleton(dc, func(dp *ioc.DependencyProvider) services2.TemplateService {
		return services2.NewTemplateService()
	})
	ioc.RegisterSingleton(dc, func(dp *ioc.DependencyProvider) services2.TokenService {
		return services2.NewTokenService()
	})
	ioc.RegisterSingleton(dc, func(dp *ioc.DependencyProvider) middlewares2.SessionService {
		return services2.NewSessionService()
	})
	ioc.RegisterSingleton(dc, func(dp *ioc.DependencyProvider) behaviours.AuditLogger {
		return services2.NewConsoleAuditLogger()
	})

	ioc.RegisterScoped(dc, func(dp *ioc.DependencyProvider) repositories2.UserRepository {
		return repositories2.NewUserRepository()
	})
	ioc.RegisterScoped(dc, func(dp *ioc.DependencyProvider) repositories2.VirtualServerRepository {
		return repositories2.NewVirtualServerRepository()
	})
	ioc.RegisterScoped(dc, func(dp *ioc.DependencyProvider) repositories2.CredentialRepository {
		return repositories2.NewCredentialRepository()
	})
	ioc.RegisterScoped(dc, func(dp *ioc.DependencyProvider) repositories2.OutboxMessageRepository {
		return repositories2.NewOutboxMessageRepository()
	})
	ioc.RegisterScoped(dc, func(dp *ioc.DependencyProvider) repositories2.FileRepository {
		return repositories2.NewFileRepository()
	})
	ioc.RegisterScoped(dc, func(dp *ioc.DependencyProvider) repositories2.TemplateRepository {
		return repositories2.NewTemplateRepository()
	})
	ioc.RegisterScoped(dc, func(dp *ioc.DependencyProvider) repositories2.RoleRepository {
		return repositories2.NewRoleRepository()
	})
	ioc.RegisterScoped(dc, func(dp *ioc.DependencyProvider) repositories2.GroupRepository {
		return repositories2.NewGroupRepository()
	})
	ioc.RegisterScoped(dc, func(dp *ioc.DependencyProvider) repositories2.GroupRoleRepository {
		return repositories2.NewGroupRoleRepository()
	})
	ioc.RegisterScoped(dc, func(dp *ioc.DependencyProvider) repositories2.UserRoleAssignmentRepository {
		return repositories2.NewUserRoleAssignmentRepository()
	})
	ioc.RegisterScoped(dc, func(dp *ioc.DependencyProvider) repositories2.ApplicationRepository {
		return repositories2.NewApplicationRepository()
	})
	ioc.RegisterScoped(dc, func(dp *ioc.DependencyProvider) repositories2.SessionRepository {
		return repositories2.NewSessionRepository()
	})

	setupMediator(dc)
	dp := dc.BuildProvider()

	jobManager := jobs.NewJobManager(jobs.WithOnError(func(err error) {
		logging.Logger.Errorf("an error happened while running a job: %v", err)
	}))

	/*jobManager.QueueJob(
		jobs.OutboxSendingJob(dp),
		time.Second,
		jobs.WithName("outbox_sender"),
		jobs.WithStartImmediate(),
	)*/

	jobManager.Start(context.Background())

	initApplication(dp)

	server.Serve(dp)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
}

func setupMediator(dc *ioc.DependencyCollection) {
	m := mediator.NewMediator()

	mediator.RegisterHandler(m, queries2.HandleAnyVirtualServerExists)
	mediator.RegisterHandler(m, queries2.HandleGetVirtualServerPublicInfo)
	mediator.RegisterHandler(m, queries2.HandleGetVirtualServerQuery)
	mediator.RegisterHandler(m, commands2.HandleCreateVirtualServer)

	mediator.RegisterHandler(m, queries2.HandleListTemplates)
	mediator.RegisterHandler(m, queries2.HandleGetTemplate)

	mediator.RegisterHandler(m, commands2.HandleRegisterUser)
	mediator.RegisterHandler(m, commands2.HandleCreateUser)
	mediator.RegisterHandler(m, commands2.HandleVerifyEmail)
	mediator.RegisterHandler(m, commands2.HandleResetPassword)
	mediator.RegisterHandler(m, queries2.HandleGetUserQuery)
	mediator.RegisterHandler(m, commands2.HandlePatchUser)
	mediator.RegisterHandler(m, queries2.HandleListUsers)

	mediator.RegisterHandler(m, commands2.HandleCreateApplication)
	mediator.RegisterHandler(m, queries2.HandleListApplications)
	mediator.RegisterHandler(m, queries2.HandleGetApplication)
	mediator.RegisterHandler(m, commands2.HandlePatchApplication)
	mediator.RegisterHandler(m, commands2.HandleDeleteApplication)

	mediator.RegisterHandler(m, queries2.HandleListRoles)
	mediator.RegisterHandler(m, queries2.HandleGetRole)
	mediator.RegisterHandler(m, commands2.HandleCreateRole)
	mediator.RegisterHandler(m, commands2.HandleAssignRoleToUser)

	mediator.RegisterEventHandler(m, events.QueueEmailVerificationJobOnUserCreatedEvent)

	mediator.RegisterBehaviour(m, behaviours.PolicyBehaviour)

	ioc.RegisterSingleton(dc, func(dp *ioc.DependencyProvider) mediator.Mediator {
		return m
	})
}

// initApplication sets up the initial application state on the first startup.
// It creates an initial virtual server and other necessary defaults if none exist.
func initApplication(dp *ioc.DependencyProvider) {
	scope := dp.NewScope()
	defer utils.PanicOnError(scope.Close, "failed creating scope to init application")

	ctx := middlewares2.ContextWithScope(context.Background(), scope)
	m := ioc.GetDependency[mediator.Mediator](scope)

	// check if there are no virtual servers
	existsResult, err := mediator.Send[*queries2.AnyVirtualServerExistsResult](ctx, m, queries2.AnyVirtualServerExists{})
	if err != nil {
		logging.Logger.Fatalf("failed to query if any virtual servers exist: %v", err)
	}

	if existsResult.Found {
		return
	}

	logging.Logger.Infof("Creating initial virtual server")

	// create initial vs
	_, err = mediator.Send[*commands2.CreateVirtualServerResponse](ctx, m, commands2.CreateVirtualServer{
		Name:               config.C.InitialVirtualServer.Name,
		DisplayName:        config.C.InitialVirtualServer.DisplayName,
		EnableRegistration: config.C.InitialVirtualServer.EnableRegistration,
		SigningAlgorithm:   config.C.InitialVirtualServer.SigningAlgorithm,
	})
	if err != nil {
		logging.Logger.Fatalf("failed to create initial virtual server: %v", err)
	}

	if config.C.InitialVirtualServer.CreateInitialAdmin {
		logging.Logger.Infof("Creating initial admin user")

		initialAdminUserInfo, err := mediator.Send[*commands2.CreateUserResponse](ctx, m, commands2.CreateUser{
			VirtualServerName: config.C.InitialVirtualServer.Name,
			DisplayName:       config.C.InitialVirtualServer.InitialAdmin.DisplayName,
			Username:          config.C.InitialVirtualServer.InitialAdmin.Username,
			Email:             config.C.InitialVirtualServer.InitialAdmin.PrimaryEmail,
			EmailVerified:     true,
		})
		if err != nil {
			logging.Logger.Fatalf("failed to create initial admin user: %v", err)
		}

		credentialRepository := ioc.GetDependency[repositories2.CredentialRepository](scope)
		initialAdminCredential := repositories2.NewCredential(initialAdminUserInfo.Id, &repositories2.CredentialPasswordDetails{
			HashedPassword: config.C.InitialVirtualServer.InitialAdmin.PasswordHash,
			Temporary:      false,
		})
		err = credentialRepository.Insert(ctx, initialAdminCredential)
		if err != nil {
			logging.Logger.Fatalf("failed to create initial admin credential: %v", err)
		}
	}
}
func configureSwaggerFromConfig() {
	if config.C.Server.ExternalUrl != "" {
		if u, err := url.Parse(config.C.Server.ExternalUrl); err == nil {
			if u.Host != "" {
				docs.SwaggerInfo.Host = u.Host
			}
			if u.Scheme != "" {
				docs.SwaggerInfo.Schemes = []string{u.Scheme}
			}
		}
	}

	// Fallback to internal host:port
	if docs.SwaggerInfo.Host == "" {
		docs.SwaggerInfo.Host = fmt.Sprintf("%s:%d", config.C.Server.Host, config.C.Server.Port)
	}
	if len(docs.SwaggerInfo.Schemes) == 0 {
		docs.SwaggerInfo.Schemes = []string{"http"}
	}

	docs.SwaggerInfo.BasePath = "/"
}
