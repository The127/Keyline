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
	"Keyline/internal/clock"
	"Keyline/internal/commands"
	"Keyline/internal/config"
	"Keyline/internal/database"
	"Keyline/internal/events"
	"Keyline/internal/jobs"
	"Keyline/internal/logging"
	"Keyline/internal/metrics"
	"Keyline/internal/middlewares"
	"Keyline/internal/queries"
	"Keyline/internal/repositories"
	"Keyline/internal/server"
	"Keyline/internal/services"
	"Keyline/ioc"
	"Keyline/mediator"
	"Keyline/utils"
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"syscall"

	"Keyline/docs"

	"github.com/huandu/go-sqlbuilder"
)

func main() {
	config.Init()
	configureSwaggerFromConfig()

	sqlbuilder.DefaultFlavor = sqlbuilder.PostgreSQL

	logging.Init()
	metrics.Init()
	database.Migrate()

	dc := ioc.NewDependencyCollection()
	ioc.RegisterSingleton(dc, func(dp *ioc.DependencyProvider) *sql.DB {
		return database.ConnectToDatabase()
	})

	ioc.RegisterScoped(dc, func(dp *ioc.DependencyProvider) database.DbService {
		return database.NewDbService(dp)
	})
	ioc.RegisterCloseHandler(dc, func(dbService database.DbService) error {
		return dbService.Close()
	})

	ioc.RegisterSingleton(dc, func(dp *ioc.DependencyProvider) clock.Service {
		return clock.NewClockService()
	})

	ioc.RegisterSingleton(dc, func(dp *ioc.DependencyProvider) services.KeyCache {
		return services.NewMemoryCache[string, services.KeyPair]()
	})
	ioc.RegisterSingleton(dc, func(dp *ioc.DependencyProvider) services.KeyStore {
		switch config.C.KeyStore.Mode {
		case config.KeyStoreModeDirectory:
			return services.NewDirectoryKeyStore()

		case config.KeyStoreModeOpenBao:
			panic("not implemented yet")

		default:
			panic("not implemented")
		}
	})
	ioc.RegisterSingleton(dc, func(dp *ioc.DependencyProvider) services.KeyService {
		return services.NewKeyService(
			ioc.GetDependency[services.KeyCache](dp),
			ioc.GetDependency[services.KeyStore](dp),
		)
	})
	ioc.RegisterSingleton(dc, func(dp *ioc.DependencyProvider) services.MailService {
		return services.NewMailService()
	})
	ioc.RegisterSingleton(dc, func(dp *ioc.DependencyProvider) services.TemplateService {
		return services.NewTemplateService()
	})
	ioc.RegisterSingleton(dc, func(dp *ioc.DependencyProvider) services.TokenService {
		return services.NewTokenService()
	})
	ioc.RegisterSingleton(dc, func(dp *ioc.DependencyProvider) middlewares.SessionService {
		return services.NewSessionService()
	})
	ioc.RegisterSingleton(dc, func(dp *ioc.DependencyProvider) behaviours.AuditLogger {
		return services.NewConsoleAuditLogger()
	})

	ioc.RegisterScoped(dc, func(dp *ioc.DependencyProvider) repositories.UserRepository {
		return repositories.NewUserRepository()
	})
	ioc.RegisterScoped(dc, func(dp *ioc.DependencyProvider) repositories.VirtualServerRepository {
		return repositories.NewVirtualServerRepository()
	})
	ioc.RegisterScoped(dc, func(dp *ioc.DependencyProvider) repositories.CredentialRepository {
		return repositories.NewCredentialRepository()
	})
	ioc.RegisterScoped(dc, func(dp *ioc.DependencyProvider) repositories.OutboxMessageRepository {
		return repositories.NewOutboxMessageRepository()
	})
	ioc.RegisterScoped(dc, func(dp *ioc.DependencyProvider) repositories.FileRepository {
		return repositories.NewFileRepository()
	})
	ioc.RegisterScoped(dc, func(dp *ioc.DependencyProvider) repositories.TemplateRepository {
		return repositories.NewTemplateRepository()
	})
	ioc.RegisterScoped(dc, func(dp *ioc.DependencyProvider) repositories.RoleRepository {
		return repositories.NewRoleRepository()
	})
	ioc.RegisterScoped(dc, func(dp *ioc.DependencyProvider) repositories.GroupRepository {
		return repositories.NewGroupRepository()
	})
	ioc.RegisterScoped(dc, func(dp *ioc.DependencyProvider) repositories.GroupRoleRepository {
		return repositories.NewGroupRoleRepository()
	})
	ioc.RegisterScoped(dc, func(dp *ioc.DependencyProvider) repositories.UserRoleAssignmentRepository {
		return repositories.NewUserRoleAssignmentRepository()
	})
	ioc.RegisterScoped(dc, func(dp *ioc.DependencyProvider) repositories.ApplicationRepository {
		return repositories.NewApplicationRepository()
	})
	ioc.RegisterScoped(dc, func(dp *ioc.DependencyProvider) repositories.SessionRepository {
		return repositories.NewSessionRepository()
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

	mediator.RegisterHandler(m, queries.HandleAnyVirtualServerExists)
	mediator.RegisterHandler(m, queries.HandleGetVirtualServerPublicInfo)
	mediator.RegisterHandler(m, queries.HandleGetVirtualServerQuery)
	mediator.RegisterHandler(m, commands.HandleCreateVirtualServer)

	mediator.RegisterHandler(m, queries.HandleListTemplates)
	mediator.RegisterHandler(m, queries.HandleGetTemplate)

	mediator.RegisterHandler(m, commands.HandleRegisterUser)
	mediator.RegisterHandler(m, commands.HandleCreateUser)
	mediator.RegisterHandler(m, commands.HandleVerifyEmail)
	mediator.RegisterHandler(m, commands.HandleResetPassword)
	mediator.RegisterHandler(m, queries.HandleGetUserQuery)
	mediator.RegisterHandler(m, commands.HandlePatchUser)
	mediator.RegisterHandler(m, queries.HandleListUsers)
	mediator.RegisterHandler(m, commands.HandleCreateServiceUser)
	mediator.RegisterHandler(m, commands.HandleAssociateServiceUserPublicKey)

	mediator.RegisterHandler(m, commands.HandleCreateApplication)
	mediator.RegisterHandler(m, queries.HandleListApplications)
	mediator.RegisterHandler(m, queries.HandleGetApplication)
	mediator.RegisterHandler(m, commands.HandlePatchApplication)
	mediator.RegisterHandler(m, commands.HandleDeleteApplication)

	mediator.RegisterHandler(m, queries.HandleListRoles)
	mediator.RegisterHandler(m, queries.HandleGetRole)
	mediator.RegisterHandler(m, commands.HandleCreateRole)
	mediator.RegisterHandler(m, commands.HandleAssignRoleToUser)
	mediator.RegisterHandler(m, queries.HandleListUsersInRole)

	mediator.RegisterHandler(m, queries.HandleListGroups)

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

	ctx := middlewares.ContextWithScope(context.Background(), scope)
	m := ioc.GetDependency[mediator.Mediator](scope)

	// check if there are no virtual servers
	existsResult, err := mediator.Send[*queries.AnyVirtualServerExistsResult](ctx, m, queries.AnyVirtualServerExists{})
	if err != nil {
		logging.Logger.Fatalf("failed to query if any virtual servers exist: %v", err)
	}

	if existsResult.Found {
		return
	}

	logging.Logger.Infof("Creating initial virtual server")

	// create initial vs
	_, err = mediator.Send[*commands.CreateVirtualServerResponse](ctx, m, commands.CreateVirtualServer{
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

		initialAdminUserInfo, err := mediator.Send[*commands.CreateUserResponse](ctx, m, commands.CreateUser{
			VirtualServerName: config.C.InitialVirtualServer.Name,
			DisplayName:       config.C.InitialVirtualServer.InitialAdmin.DisplayName,
			Username:          config.C.InitialVirtualServer.InitialAdmin.Username,
			Email:             config.C.InitialVirtualServer.InitialAdmin.PrimaryEmail,
			EmailVerified:     true,
		})
		if err != nil {
			logging.Logger.Fatalf("failed to create initial admin user: %v", err)
		}

		credentialRepository := ioc.GetDependency[repositories.CredentialRepository](scope)
		initialAdminCredential := repositories.NewCredential(initialAdminUserInfo.Id, &repositories.CredentialPasswordDetails{
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
	} else {
		docs.SwaggerInfo.Host = fmt.Sprintf("%s:%d", config.C.Server.Host, config.C.Server.Port)
	}

	if len(docs.SwaggerInfo.Schemes) == 0 {
		docs.SwaggerInfo.Schemes = []string{"http"}
	}

	docs.SwaggerInfo.BasePath = "/"
}
