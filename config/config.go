package config

import "flag"

type Config struct {
	Server struct {
		Host string
		Port int
	}
}

var configFilePath string
var C Config

func Init() {
	// read flags (read config file path)
	readFlags()
	println(configFilePath)

	// set default values
	// read values from different sources (env vars & files)
	// validate config
	// set the global variable
}

func readFlags() {
	// read flags passed to the program
	flag.StringVar(&configFilePath, "config", "./config.yaml", "The path for the config file.")
	flag.Parse()
}
