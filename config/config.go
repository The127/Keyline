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
}

var configFilePath string
var C Config

func Init() {
	// read flags (read config file path)
	readFlags()

	// set default values

	// read values from different sources (env vars & files)
	readConfigFile()

	// validate config
	// set the global variable
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
	flag.Parse()
}
