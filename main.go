package main

import (
	"Keyline/config"
	"Keyline/database"
	"Keyline/ioc"
	"Keyline/logging"
	"Keyline/mediator"
	"Keyline/server"
)

func main() {
	config.Init()
	logging.Init()
	database.Migrate()

	dc := ioc.NewDependencyCollection()
	setupMediator(dc)
	dp := dc.BuildProvider()

	initApplication()

	server.Serve(dp)
}

func setupMediator(dc *ioc.DependencyCollection) {
	m := mediator.NewMediator()

	ioc.RegisterSingleton(dc, func() *mediator.Mediator {
		return m
	})
}

// initApplication sets up an initial application on first startup
// it creates an initial virtual server and other stuff
func initApplication() {
	// check if there are no virtual servers

	// create initial vs
}
