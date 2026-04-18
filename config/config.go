package config

import (
	"github.com/The127/Keyline/internal/retry"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env/v2"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

// QueueMode has the following constants: DeliveryModeNoop, DeliveryModeInProcess
type QueueMode string

const (
	QueueModeNoop      QueueMode = "noop"
	QueueModeInProcess QueueMode = "in-process"
)

// DatabaseMode has the following constants: DatabaseModePostgres, DatabaseModeSqlite
type DatabaseMode string

const (
	DatabaseModePostgres DatabaseMode = "postgres"
	DatabaseModeSqlite   DatabaseMode = "sqlite"
	DatabaseModeMemory   DatabaseMode = "memory"
)

// CacheMode has the following constants: CacheModeMemory, CacheModeRedis
type CacheMode string

const (
	CacheModeMemory CacheMode = "memory"
	CacheModeRedis  CacheMode = "redis"
)

// KeyStoreMode has the following constants: KeyStoreModeMemory (testing only), KeyStoreModeDirectory, KeyStoreModeOpenBao
type KeyStoreMode string

const (
	KeyStoreModeMemory    KeyStoreMode = "memory"
	KeyStoreModeDirectory KeyStoreMode = "directory"
	KeyStoreModeVault     KeyStoreMode = "vault"
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
	Server   ServerConfig `yaml:"server"`
	Frontend struct {
		ExternalUrl string `yaml:"externalUrl"`
	} `yaml:"frontend"`
	Database             DatabaseConfig             `yaml:"database"`
	InitialVirtualServer InitialVirtualServerConfig `yaml:"initialVirtualServer"`
	KeyStore             KeyStoreConfig             `yaml:"keyStore"`
	Cache                struct {
		Mode  CacheMode `yaml:"mode"`
		Redis struct {
			Host     string `yaml:"host"`
			Port     int    `yaml:"port"`
			Username string `yaml:"username"`
			Password string `yaml:"password"`
			Database int    `yaml:"database"`
		} `yaml:"redis"`
	} `yaml:"cache"`
	LeaderElection LeaderElectionConfig `yaml:"leaderElection"`
}

type InitialVirtualServerConfig struct {
	Name                  string           `yaml:"name"`
	DisplayName           string           `yaml:"displayName"`
	EnableRegistration    bool             `yaml:"enableRegistration"`
	SigningAlgorithm      SigningAlgorithm  `yaml:"signingAlgorithm"`
	CreateSystemAdminRole bool             `yaml:"createSystemAdminRole"`
	CreateAdmin           bool             `yaml:"createAdmin"`
	Admin                 struct {
		Username     string   `yaml:"username"`
		DisplayName  string   `yaml:"displayName"`
		PrimaryEmail string   `yaml:"primaryEmail"`
		PasswordHash string   `yaml:"passwordHash"`
		Roles        []string `yaml:"roles"`
	} `yaml:"admin"`
	ServiceUsers []ServiceUserConfig    `yaml:"serviceUsers"`
	Projects     []InitialProjectConfig `yaml:"projects"`
	Mail         struct {
		Host     string `yaml:"host"`
		Port     int    `yaml:"port"`
		Username string `yaml:"username"`
		Password string `yaml:"password"`
	} `yaml:"mail"`
}

type ServiceUserConfig struct {
	Username  string `yaml:"username"`
	Roles     []string `yaml:"roles"`
	PublicKey struct {
		Pem string `yaml:"pem"`
		Kid string `yaml:"kid"`
	} `yaml:"publicKey"`
}

type KeyStoreConfig struct {
	Mode      KeyStoreMode          `yaml:"mode"`
	Vault     VaultKeyStoreConfig   `yaml:"vault"`
	Directory struct {
		Path string `yaml:"path"`
	} `yaml:"directory"`
}

type VaultKeyStoreConfig struct {
	Address string `yaml:"address"`
	Token   string `yaml:"token"`
	Mount   string `yaml:"mount"`
	Prefix  string `yaml:"prefix,omitempty"`
}

type LeaderElectionMode string

const (
	LeaderElectionModeNone LeaderElectionMode = "none"
	LeaderElectionModeRaft LeaderElectionMode = "raft"
)

type InitialProjectConfig struct {
	Slug        string `yaml:"slug"`
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Roles       []struct {
		Name        string `yaml:"name"`
		Description string `yaml:"description"`
	} `yaml:"roles"`
	Applications []struct {
		Name                   string   `yaml:"name"`
		DisplayName            string   `yaml:"displayName"`
		Type                   string   `yaml:"type"`
		HashedSecret           *string  `yaml:"hashedSecret,omitempty"`
		RedirectUris           []string `yaml:"redirectUris"`
		PostLogoutRedirectUris []string `yaml:"postLogoutRedirectUris"`
		DeviceFlowEnabled      bool     `yaml:"deviceFlowEnabled"`
	} `yaml:"applications"`
	ResourceServers []struct {
		Slug        string `yaml:"slug"`
		Name        string `yaml:"name"`
		Description string `yaml:"description"`
	} `yaml:"resourceServers"`
}

type LeaderElectionConfig struct {
	Mode LeaderElectionMode     `yaml:"mode"`
	Raft LeaderElectionRaftConfig `yaml:"raft"`
}

type LeaderElectionRaftConfig struct {
	Host        string     `yaml:"host"`
	Port        int        `yaml:"port"`
	Id          string     `yaml:"id"`
	InitiatorId string     `yaml:"initiatorId"`
	Nodes       []RaftNode `yaml:"nodes"`
}

type RaftNode struct {
	Id      string `yaml:"id"`
	Address string `yaml:"address"`
}

type ServerConfig struct {
	ExternalUrl    string   `yaml:"externalUrl"`
	ExternalDomain string   `yaml:"externalDomain"`
	Host           string   `yaml:"host"`
	Port           int      `yaml:"port"`
	ApiPort        int      `yaml:"apiPort"`
	AllowedOrigins []string `yaml:"allowedOrigins"`
}

type DatabaseConfig struct {
	Mode     DatabaseMode   `yaml:"mode"`
	Postgres PostgresConfig `yaml:"postgres"`
	Sqlite   struct {
		Database string `yaml:"database"`
	} `yaml:"sqlite"`
}

type PostgresConfig struct {
	Database string `yaml:"database"`
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	SslMode  string `yaml:"sslMode"`
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
		retry.FiveTimes(func() error {
			_, err := os.Stat(configFilePath)
			if err != nil {
				return fmt.Errorf("failed to stat config file: %w", err)
			}

			return nil
		}, "failed to read config file")

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
	setLeaderElectionDefaultsOrPanic()
}

func setLeaderElectionDefaultsOrPanic() {
	switch C.LeaderElection.Mode {
	case LeaderElectionModeNone:
		// nothing to do

	case LeaderElectionModeRaft:
		setLeaderElectionRaftDefaultsOrPanic()

	default:
		panic("leader election mode missing or not supported")
	}
}

func setLeaderElectionRaftDefaultsOrPanic() {
	if C.LeaderElection.Raft.Host == "" {
		panic("missing leader election raft host")
	}

	if C.LeaderElection.Raft.Port == 0 {
		panic("missing leader election raft port")
	}

	if C.LeaderElection.Raft.Id == "" {
		panic("missing leader election raft id")
	}

	if C.LeaderElection.Raft.InitiatorId == "" {
		panic("missing leader election raft initiator id")
	}

	if len(C.LeaderElection.Raft.Nodes) == 0 {
		panic("missing leader election raft nodes")
	}

	for _, node := range C.LeaderElection.Raft.Nodes {
		if node.Id == "" {
			panic("missing leader election raft node id")
		}

		if node.Address == "" {
			panic("missing leader election raft node address")
		}
	}
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
	case KeyStoreModeMemory:
		// nothing to do

	case KeyStoreModeVault:
		setKeyStoreModeVaultDefaultsOrPanic()

	case KeyStoreModeDirectory:
		setKeyStoreModeDirectoryDefaultsOrPanic()

	default:
		panic("key store mode missing or not supported")
	}
}

func setKeyStoreModeVaultDefaultsOrPanic() {
	if C.KeyStore.Vault.Address == "" {
		panic("missing key store vault address")
	}

	if C.KeyStore.Vault.Token == "" {
		panic("missing key store vault token")
	}

	if C.KeyStore.Vault.Mount == "" {
		panic("missing key store vault mount")
	}
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
	setInitialServiceUserDefaultsOrPanic()
	setInitialProjectsDefaultsOrPanic()
}

func setInitialServiceUserDefaultsOrPanic() {
	for _, serviceUser := range C.InitialVirtualServer.ServiceUsers {
		if serviceUser.Username == "" {
			panic("missing service user username")
		}

		if serviceUser.PublicKey.Pem == "" {
			panic("missing service user public key pem")
		}

		if serviceUser.PublicKey.Kid == "" {
			panic("missing service user public key kid")
		}

	}
}

func setInitialProjectsDefaultsOrPanic() {
	for i := range C.InitialVirtualServer.Projects {
		project := &C.InitialVirtualServer.Projects[i]

		if project.Slug == "" {
			panic("missing project slug")
		}

		if project.Name == "" {
			project.Name = project.Slug
		}

		setInitialApplicationsDefaultsOrPanic(project)
		setInitialRolesDefaultsOrPanic(project)
		setInitialResourceServersDefaultsOrPanic(project)
	}
}

func setInitialResourceServersDefaultsOrPanic(project *InitialProjectConfig) {
	for i := range project.ResourceServers {
		resourceServer := &project.ResourceServers[i]

		if resourceServer.Slug == "" {
			panic("missing resource server slug")
		}
		if resourceServer.Name == "" {
			resourceServer.Name = resourceServer.Slug
		}
		if resourceServer.Description == "" {
			resourceServer.Description = resourceServer.Name
		}
	}
}

func setInitialRolesDefaultsOrPanic(project *InitialProjectConfig) {
	for i := range project.Roles {
		role := &project.Roles[i]

		if role.Name == "" {
			panic("missing role name")
		}
	}
}

func setInitialApplicationsDefaultsOrPanic(project *InitialProjectConfig) {
	for i := range project.Applications {
		application := &project.Applications[i]

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
	if !C.InitialVirtualServer.CreateAdmin {
		return
	}

	if C.InitialVirtualServer.Admin.Username == "" {
		C.InitialVirtualServer.Admin.Username = "admin"
	}

	if C.InitialVirtualServer.Admin.DisplayName == "" {
		C.InitialVirtualServer.Admin.DisplayName = "Administrator"
	}

	if C.InitialVirtualServer.Admin.PrimaryEmail == "" {
		panic("missing initial admin primary email")
	}

	if C.InitialVirtualServer.Admin.PasswordHash == "" {
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

	if C.Server.ExternalDomain == "" {
		externalUrl, err := url.Parse(C.Server.ExternalUrl)
		if err != nil {
			panic(fmt.Errorf("extracting domain from external url: %w", err))
		}
		C.Server.ExternalDomain = externalUrl.Hostname()
	}

	if C.Server.Port == 0 {
		C.Server.Port = 8080
	}

	if C.Server.ApiPort == C.Server.Port {
		panic("api port must be different from server port (server port defaults to 8080)")
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

	case DatabaseModeMemory:
		// no-op: in-memory database has no connection parameters

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
