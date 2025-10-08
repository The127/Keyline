package config

import (
	"flag"
	"fmt"

	"github.com/spf13/viper"
)

// KeyStoreMode has the following constants: KeyStoreModeDirectory, KeyStoreModeOpenBao
type KeyStoreMode string

const (
	KeyStoreModeDirectory KeyStoreMode = "directory"
	KeyStoreModeOpenBao   KeyStoreMode = "openbao"
)

type SigningAlgorithm string

const (
	SigningAlgorithmRS256 SigningAlgorithm = "RS256"
	SigningAlgorithmEdDSA SigningAlgorithm = "EdDSA"
)

type Config struct {
	Server struct {
		ExternalUrl    string
		Host           string
		Port           int
		AllowedOrigins []string
	}
	Frontend struct {
		ExternalUrl string
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
		SigningAlgorithm   SigningAlgorithm
		CreateInitialAdmin bool
		InitialAdmin       struct {
			Username     string
			DisplayName  string
			PrimaryEmail string
			PasswordHash string
		}
		Mail struct {
			Host     string
			Port     int
			Username string
			Password string
		}
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
	Redis struct {
		Host     string
		Port     int
		Username string
		Password string
		Database int
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

	// read values from different sources (env vars & files)
	readConfigFile()
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
	setFrontendDefaultsOrPanic()
	setDatabaseDefaultsOrPanic()
	setInitialVirtualServerDefaultsOrPanic()
	setKeyStoreDefaultsOrPanic()
	setRedisDefaultsOrPanic()
}

func setFrontendDefaultsOrPanic() {
	if C.Frontend.ExternalUrl == "" {
		if IsProduction() {
			panic("missing frontend external url")
		}
		C.Frontend.ExternalUrl = "http://localhost:5173"
	}
}

func setRedisDefaultsOrPanic() {
	if C.Redis.Host == "" {
		if IsProduction() {
			panic("missing redis host")
		}

		C.Redis.Host = "localhost"
	}

	if C.Redis.Port == 0 {
		C.Redis.Port = 6379
	}
}

func setKeyStoreDefaultsOrPanic() {
	switch C.KeyStore.Mode {
	case KeyStoreModeOpenBao:
		setKeyStoreModeOpenBaoDefaultsOrPanic()

	case KeyStoreModeDirectory:
		setKeyStoreModeDirectoryDefaultsOrPanic()

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

	if C.InitialVirtualServer.SigningAlgorithm == "" {
		C.InitialVirtualServer.SigningAlgorithm = SigningAlgorithmEdDSA
	}

	setInitialAdminDefaultsOrPanic()
}

func setInitialAdminDefaultsOrPanic() {
	if !C.InitialVirtualServer.CreateInitialAdmin {
		return
	}

	if C.InitialVirtualServer.InitialAdmin.Username == "" {
		C.InitialVirtualServer.InitialAdmin.Username = "admin"
	}

	if C.InitialVirtualServer.InitialAdmin.DisplayName == "" {
		C.InitialVirtualServer.InitialAdmin.DisplayName = "Administrator"
	}

	if C.InitialVirtualServer.InitialAdmin.PrimaryEmail == "" {
		panic("missing initial admin primary email")
	}

	if C.InitialVirtualServer.InitialAdmin.PasswordHash == "" {
		panic("missing initial admin password hash")
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
		C.Server.Port = 8080
	}

	if C.Server.ExternalUrl == "" {
		if IsProduction() {
			panic("missing external url")
		}

		C.Server.ExternalUrl = fmt.Sprintf("%s:%d", C.Server.Host, C.Server.Port)
	}

	if len(C.Server.AllowedOrigins) == 0 {
		if IsProduction() {
			panic("missing allowed origins")
		}

		C.Server.AllowedOrigins = []string{"*", "http://localhost:5173"}
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
