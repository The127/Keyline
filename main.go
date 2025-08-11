package main

import (
	"Keyline/commands"
	"Keyline/config"
	"Keyline/database"
	"Keyline/ioc"
	"Keyline/logging"
	"Keyline/mediator"
	"Keyline/middlewares"
	"Keyline/queries"
	"Keyline/server"
	"Keyline/services"
	"Keyline/utils"
	"context"
	"database/sql"
)

func main() {
	config.Init()
	logging.Init()
	database.Migrate()

	dc := ioc.NewDependencyCollection()
	ioc.RegisterSingleton(dc, func(dp *ioc.DependencyProvider) *sql.DB {
		return database.ConnectToDatabase()
	})

	ioc.RegisterScoped(dc, func(dp *ioc.DependencyProvider) *services.DbService {
		return services.NewDbService(dp)
	})
	ioc.RegisterCloseHandler(dc, func(dbService *services.DbService) error {
		return dbService.Close()
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

	// create initial vs
	_, err = mediator.Send[*commands.CreateVirtualServerResponse](ctx, m, commands.CreateVirtualServer{
		Name:        "default",
		DisplayName: "Default Virtual Server",
	})
	if err != nil {
		logging.Logger.Fatalf("failed to create intial virtual server: %v", err)
	}
}
