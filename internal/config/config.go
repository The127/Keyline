package config

import (
	"flag"
	"fmt"
	"log"
	"net/url"
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
	Server   ServerConfig
	Frontend struct {
		ExternalUrl string
	}
	Database struct {
		Mode     DatabaseMode
		Postgres PostgresConfig
		Sqlite   struct {
			Database string
		}
	}
	InitialVirtualServer struct {
		Name               string
		DisplayName        string
		EnableRegistration bool
		SigningAlgorithm   SigningAlgorithm
		CreateAdmin        bool
		Admin              struct {
			Username     string
			DisplayName  string
			PrimaryEmail string
			PasswordHash string
		}
		ServiceUsers []struct {
			Username  string
			Roles     []string
			PublicKey string
		}
		Projects []InitialProjectConfig
		Mail     struct {
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
	LeaderElection LeaderElectionConfig
}

type LeaderElectionMode string

const (
	LeaderElectionModeNone LeaderElectionMode = "none"
	LeaderElectionModeRaft LeaderElectionMode = "raft"
)

type InitialProjectConfig struct {
	Slug        string
	Name        string
	Description string
	Roles       []struct {
		Name        string
		Description string
	}
	Applications []struct {
		Name                   string
		DisplayName            string
		Type                   string
		HashedSecret           *string
		RedirectUris           []string
		PostLogoutRedirectUris []string
	}
	ResourceServers []struct {
		Name        string
		Description string
	}
}

type LeaderElectionConfig struct {
	Mode LeaderElectionMode
	Raft LeaderElectionRaftConfig
}

type LeaderElectionRaftConfig struct {
	Host        string
	Port        int
	Id          string
	InitiatorId string
	Nodes       []RaftNode
}

type RaftNode struct {
	Id      string
	Address string
}

type ServerConfig struct {
	ExternalUrl    string
	ExternalDomain string
	Host           string
	Port           int
	AllowedOrigins []string
}

type PostgresConfig struct {
	Database string
	Host     string
	Port     int
	Username string
	Password string
	SslMode  string
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
	setInitialServiceUserDefaultsOrPanic()
	setInitialProjectsDefaultsOrPanic()
}

func setInitialServiceUserDefaultsOrPanic() {
	for _, serviceUser := range C.InitialVirtualServer.ServiceUsers {
		if serviceUser.Username == "" {
			panic("missing service user username")
		}

		if serviceUser.PublicKey == "" {
			panic("missing service user public key")
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

		if resourceServer.Name == "" {
			panic("missing resource server name")
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
