package main

import (
	"Keyline/commands"
	"Keyline/config"
	"Keyline/database"
	"Keyline/events"
	"Keyline/ioc"
	"Keyline/logging"
	"Keyline/mediator"
	"Keyline/middlewares"
	"Keyline/queries"
	"Keyline/repositories"
	"Keyline/server"
	"Keyline/services"
	"Keyline/utils"
	"context"
	"database/sql"
	"github.com/huandu/go-sqlbuilder"
)

func main() {
	sqlbuilder.DefaultFlavor = sqlbuilder.PostgreSQL

	config.Init()
	logging.Init()
	database.Migrate()

	dc := ioc.NewDependencyCollection()
	ioc.RegisterSingleton(dc, func(dp *ioc.DependencyProvider) *sql.DB {
		return database.ConnectToDatabase()
	})

	ioc.RegisterScoped(dc, func(dp *ioc.DependencyProvider) *database.DbService {
		return database.NewDbService(dp)
	})
	ioc.RegisterCloseHandler(dc, func(dbService *database.DbService) error {
		return dbService.Close()
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

	ioc.RegisterScoped(dc, func(dp *ioc.DependencyProvider) *repositories.UserRepository {
		return &repositories.UserRepository{}
	})
	ioc.RegisterScoped(dc, func(dp *ioc.DependencyProvider) *repositories.VirtualServerRepository {
		return &repositories.VirtualServerRepository{}
	})
	ioc.RegisterScoped(dc, func(dp *ioc.DependencyProvider) *repositories.CredentialRepository {
		return &repositories.CredentialRepository{}
	})
	ioc.RegisterScoped(dc, func(dp *ioc.DependencyProvider) *repositories.OutboxMessageRepository {
		return &repositories.OutboxMessageRepository{}
	})
	ioc.RegisterScoped(dc, func(dp *ioc.DependencyProvider) *repositories.FileRepository {
		return &repositories.FileRepository{}
	})
	ioc.RegisterScoped(dc, func(dp *ioc.DependencyProvider) *repositories.TemplateRepository {
		return &repositories.TemplateRepository{}
	})

	setupMediator(dc)
	dp := dc.BuildProvider()

	initApplication(dp)

	server.Serve(dp)
}

func setupMediator(dc *ioc.DependencyCollection) {
	m := mediator.NewMediator()

	mediator.RegisterHandler(m, queries.HandleAnyVirtualServerExists)

	mediator.RegisterHandler(m, commands.HandleCreateVirtualServer)
	mediator.RegisterHandler(m, commands.HandleRegisterUser)

	mediator.RegisterEventHandler(m, events.QueueEmailVerificationJobOnUserCreatedEvent)

	ioc.RegisterSingleton(dc, func(dp *ioc.DependencyProvider) *mediator.Mediator {
		return m
	})
}

// initApplication sets up the initial application state on first startup.
// It creates an initial virtual server and other necessary defaults if none exist.
func initApplication(dp *ioc.DependencyProvider) {
	scope := dp.NewScope()
	defer utils.PanicOnError(scope.Close, "failed creating scope to init application")

	ctx := middlewares.ContextWithScope(context.Background(), scope)
	m := ioc.GetDependency[*mediator.Mediator](scope)

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
	})
	if err != nil {
		logging.Logger.Fatalf("failed to create intial virtual server: %v", err)
	}
}
