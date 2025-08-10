package main

import (
	"Keyline/config"
	"Keyline/database"
	"Keyline/ioc"
	"Keyline/logging"
	"Keyline/mediator"
	"Keyline/queries/virtualServers"
	"Keyline/server"
	"Keyline/services"
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

	initApplication()

	server.Serve(dp)
}

func setupMediator(dc *ioc.DependencyCollection) {
	m := mediator.NewMediator()

	mediator.RegisterHandler(m, virtualServers.DoesAnyVirtualServerExistQueryHandler)

	ioc.RegisterSingleton(dc, func(dp *ioc.DependencyProvider) *mediator.Mediator {
		return m
	})
}

// initApplication sets up an initial application on first startup
// it creates an initial virtual server and other stuff
func initApplication() {
	// check if there are no virtual servers

	// create initial vs
}
