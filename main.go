package main

import (
	"Keyline/config"
	"Keyline/logging"
	"Keyline/server"
)

func main() {
	config.Init()
	logging.Init()

	server.Serve()
}
