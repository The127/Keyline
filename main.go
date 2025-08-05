package main

import (
	"Keyline/config"
	"Keyline/server"
)

func main() {
	config.Init()

	server.Serve()
}
