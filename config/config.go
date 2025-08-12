package config

import (
	"flag"
	"github.com/spf13/viper"
)

type Config struct {
	Server struct {
		Host string
		Port int
	}
	Database struct {
		Database string
		Host     string
		Port     int
		Username string
		Password string
		SslMode  string
	}
	InitialVirtualServer struct {
		Name               string
		DisplayName        string
		EnableRegistration bool
	}
}

var configFilePath string
var environment string
var C Config

func IsProduction() bool {
	return environment == "PRODUCTION"
}

func Init() {
	// read flags (read config file path)
	readFlags()

	// TODO: set default values

	// read values from different sources (env vars & files)
	readConfigFile()

	// TODO: validate config
	// TODO: set the global variable
}

func readConfigFile() {
	v := viper.NewWithOptions(viper.KeyDelimiter("_"))

	v.SetEnvPrefix("KEYLINE")
	v.AutomaticEnv()

	v.SetConfigFile(configFilePath)

	err := v.ReadInConfig()
	if err != nil {
		panic(err)
	}

	err = v.Unmarshal(&C)
	if err != nil {
		panic(err)
	}
}

func readFlags() {
	// read flags passed to the program
	flag.StringVar(&configFilePath, "config", "./config.yaml", "The path for the config file.")
	flag.StringVar(&environment, "environment", "PRODUCTION", "The environment that this application is running in (can be PRODUCTION or DEVELOPMENT).")
	flag.Parse()
}
