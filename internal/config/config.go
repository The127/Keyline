package config

import (
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env/v2"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

// DatabaseMode has the following constants: DatabaseModePostgres, DatabaseModeSqlite
type DatabaseMode string

const (
	DatabaseModePostgres DatabaseMode = "postgres"
	DatabaseModeSqlite   DatabaseMode = "sqlite"
)

// CacheMode has the following constants: CacheModeMemory, CacheModeRedis
type CacheMode string

const (
	CacheModeMemory CacheMode = "memory"
	CacheModeRedis  CacheMode = "redis"
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

var SupportedSigningAlgorithms = []SigningAlgorithm{
	SigningAlgorithmEdDSA,
	SigningAlgorithmRS256,
}

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
		Mode     DatabaseMode
		Postgres struct {
			Database string
			Host     string
			Port     int
			Username string
			Password string
			SslMode  string
		}
		Sqlite struct {
			Database string
		}
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
		InitialApplications []struct {
			Name                   string
			DisplayName            string
			Type                   string
			HashedSecret           *string
			RedirectUris           []string
			PostLogoutRedirectUris []string
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
	Cache struct {
		Mode  CacheMode
		Redis struct {
			Host     string
			Port     int
			Username string
			Password string
			Database int
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

	// read values from different sources (env vars & files)
	readConfigFile()
}

var k = koanf.New(".")

func readConfigFile() {
	if configFilePath != "" {
		if err := k.Load(file.Provider(configFilePath), yaml.Parser()); err != nil {
			log.Fatalf("error loading config from file: %v", err)
		}
	}

	err := k.Load(env.Provider(".", env.Opt{
		Prefix: "KEYLINE_",
		TransformFunc: func(k, v string) (string, any) {
			// Transform the key.
			k = strings.ReplaceAll(strings.ToLower(strings.TrimPrefix(k, "KEYLINE_")), "_", ".")

			if strings.Contains(v, " ") {
				return k, strings.Split(v, " ")
			}

			return k, v
		},
	}), nil)
	if err != nil {
		log.Fatalf("error loading config from env: %v", err)
	}

	err = k.Unmarshal("", &C)
	if err != nil {
		log.Fatalf("error unmarshalling config: %v", err)
	}

	setDefaultsOrPanic()
}

func setDefaultsOrPanic() {
	setServerDefaultsOrPanic()
	setFrontendDefaultsOrPanic()
	setDatabaseDefaultsOrPanic()
	setInitialVirtualServerDefaultsOrPanic()
	setKeyStoreDefaultsOrPanic()
	setCacheDefaultsOrPanic()
}

func setFrontendDefaultsOrPanic() {
	if C.Frontend.ExternalUrl == "" {
		if IsProduction() {
			panic("missing frontend external url")
		}
		C.Frontend.ExternalUrl = "http://localhost:5173"
	}
}

func setCacheDefaultsOrPanic() {
	switch C.Cache.Mode {
	case CacheModeMemory:
		// nothing to do
		break

	case CacheModeRedis:
		setRedisDefaultsOrPanic()

	default:
		panic("cache mode missing or not supported")
	}
}

func setRedisDefaultsOrPanic() {
	if C.Cache.Redis.Host == "" {
		if IsProduction() {
			panic("missing redis host")
		}

		C.Cache.Redis.Host = "localhost"
	}

	if C.Cache.Redis.Port == 0 {
		C.Cache.Redis.Port = 6379
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
	setInitialApplicationsDefaultsOrPanic()
}

func setInitialApplicationsDefaultsOrPanic() {
	for i := range C.InitialVirtualServer.InitialApplications {
		application := &C.InitialVirtualServer.InitialApplications[i]

		if application.Name == "" {
			panic("missing application name")
		}

		if application.DisplayName == "" {
			application.DisplayName = application.Name
		}

		if application.Type == "" {
			panic("missing application type (confidential or public)")
		}

		if application.Type != "confidential" && application.Type != "public" {
			panic("application type not supported")
		}

		if application.Type == "confidential" && application.HashedSecret == nil {
			panic("missing application secret")
		}

		if application.Type == "confidential" && application.HashedSecret != nil {
			if len(*application.HashedSecret) == 0 {
				panic("application secret is empty")
			}
		}
	}
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
	switch C.Database.Mode {
	case DatabaseModePostgres:
		setPostgresDefaultsOrPanic()

	case DatabaseModeSqlite:
		setSqliteDefaultsOrPanic()

	default:
		panic("database mode missing or not supported")
	}
}

func setSqliteDefaultsOrPanic() {
	if C.Database.Sqlite.Database == "" {
		panic("missing sqlite file path")
	}

	panic("sqlite not implemented yet")
}

func setPostgresDefaultsOrPanic() {

	if C.Database.Postgres.Database == "" {
		C.Database.Postgres.Database = "keyline"
	}

	if C.Database.Postgres.Username == "" {
		panic("missing postgres username")
	}

	if C.Database.Postgres.Port == 0 {
		C.Database.Postgres.Port = 5432
	}

	if C.Database.Postgres.Host == "" {
		panic("missing postgres host")
	}

	if C.Database.Postgres.SslMode == "" {
		C.Database.Postgres.SslMode = "enable"
	}

	if C.Database.Postgres.Password == "" {
		panic("missing postgres password")
	}
}

func readFlags() {
	// read flags passed to the program
	flag.StringVar(&configFilePath, "config", "", "The path for the config file.")
	flag.StringVar(&environment, "environment", "PRODUCTION", "The environment that this application is running in (can be PRODUCTION or DEVELOPMENT).")
	flag.Parse()
}
