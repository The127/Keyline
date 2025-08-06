package main

import (
	"Keyline/config"
	"Keyline/database"
	"Keyline/logging"
	"Keyline/server"
)

func main() {
	config.Init()
	logging.Init()
	database.Migrate()

	server.Serve()
}
