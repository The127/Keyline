package config

import (
	"flag"
	"github.com/spf13/viper"
)

// KeyStoreMode has the following constants: KeyStoreModeDirectory, KeyStoreModeOpenBao
type KeyStoreMode string

const (
	KeyStoreModeDirectory KeyStoreMode = "directory"
	KeyStoreModeOpenBao   KeyStoreMode = "openbao"
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
	KeyStore struct {
		Mode    KeyStoreMode
		OpenBao struct {
			//TODO:
		}
		Directory struct {
			Path string
		}
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

	setDefaultsOrPanic()
}

func setDefaultsOrPanic() {
	setServerDefaultsOrPanic()
	setDatabaseDefaultsOrPanic()
	setInitialVirtualServerDefaultsOrPanic()
	setKeyStoreDefaultsOrPanic()
}

func setKeyStoreDefaultsOrPanic() {
	switch C.KeyStore.Mode {
	case KeyStoreModeOpenBao:
		setKeyStoreModeOpenBaoDefaultsOrPanic()
		break

	case KeyStoreModeDirectory:
		setKeyStoreModeDirectoryDefaultsOrPanic()
		break

	default:
		panic("key store mode missing or not supported")
	}
}

func setKeyStoreModeOpenBaoDefaultsOrPanic() {
	// TODO: implement me
	panic("not implemented")
}

func setKeyStoreModeDirectoryDefaultsOrPanic() {
	if C.KeyStore.Directory.Path == "" {
		panic("missing key store directory path")
	}
}

func setInitialVirtualServerDefaultsOrPanic() {
	if C.InitialVirtualServer.Name == "" {
		C.InitialVirtualServer.Name = "keyline"
	}

	if C.InitialVirtualServer.DisplayName == "" {
		C.InitialVirtualServer.DisplayName = "Keyline"
	}
}

func setServerDefaultsOrPanic() {
	if C.Server.Host == "" {
		if IsProduction() {
			panic("missing server hostname in config")
		}

		C.Server.Host = "localhost"
	}

	if C.Server.Port == 0 {
		C.Server.Port = 8081
	}
}

func setDatabaseDefaultsOrPanic() {
	if C.Database.Database == "" {
		C.Database.Database = "keyline"
	}

	if C.Database.Username == "" {
		panic("missing database username")
	}

	if C.Database.Port == 0 {
		C.Database.Port = 5432
	}

	if C.Database.Host == "" {
		panic("missing database host")
	}

	if C.Database.SslMode == "" {
		C.Database.SslMode = "enable"
	}

	if C.Database.Password == "" {
		panic("missing database password")
	}
}

func readFlags() {
	// read flags passed to the program
	flag.StringVar(&configFilePath, "config", "./config.yaml", "The path for the config file.")
	flag.StringVar(&environment, "environment", "PRODUCTION", "The environment that this application is running in (can be PRODUCTION or DEVELOPMENT).")
	flag.Parse()
}
