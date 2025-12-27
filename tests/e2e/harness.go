package e2e

import (
	"Keyline/client"
	"Keyline/internal/authentication"
	"Keyline/internal/commands"
	"Keyline/internal/config"
	"Keyline/internal/database"
	"Keyline/internal/database/postgres"
	"Keyline/internal/middlewares"
	"Keyline/internal/repositories"
	"Keyline/internal/server"
	"Keyline/internal/setup"
	"Keyline/utils"
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/The127/go-clock"
	"github.com/The127/ioc"
	"github.com/The127/mediatr"

	"github.com/google/uuid"
	"github.com/huandu/go-sqlbuilder"
	"golang.org/x/oauth2"
)

const (
	serviceUserUsername   = "test-service-user"
	serviceUserKid        = "7cae8bb1-71b5-4394-b45f-be5ffe81c64f"
	serviceUserPublicKey  = "-----BEGIN PUBLIC KEY-----\nMCowBQYDK2VwAyEA3M7NYNpucIwsMNDHPswe1yvLtMzIau2ddMB2FX40few=\n-----END PUBLIC KEY-----\n"
	serviceUserPrivateKey = "-----BEGIN PRIVATE KEY-----\nMFECAQEwBQYDK2VwBCIEIDlOHCg/gH43TB4S1n/2g33iti99sEkEFYwVdAkyKoqw\ngSEA3M7NYNpucIwsMNDHPswe1yvLtMzIau2ddMB2FX40few=\n-----END PRIVATE KEY-----\n"
)

type harness struct {
	c         client.Client
	ctx       context.Context
	setTime   clock.TimeSetterFn
	dbName    string
	scope     *ioc.DependencyProvider
	serverUrl string
}

func (h *harness) SetTime(t time.Time) {
	h.setTime(t)
}

func (h *harness) VirtualServer() string {
	return "test-vs"
}

func (h *harness) Ctx() context.Context {
	return h.ctx
}

func (h *harness) Client() client.Client {
	return h.c
}

func (h *harness) Close() {
	// cleanup database
	dbConnection := ioc.GetDependency[database.Database](h.scope)
	utils.PanicOnError(h.scope.Close, "closing scope")
	utils.PanicOnError(dbConnection.Close, "closing db connection in test")

	pc := config.PostgresConfig{
		Database: "postgres",
		Host:     "localhost",
		Port:     5732,
		Username: "user",
		Password: "password",
		SslMode:  "disable",
	}

	db, err := postgres.ConnectToDatabase(pc)
	if err != nil {
		panic(err)
	}
	defer utils.PanicOnError(db.Close, "closing initial db connection in test")

	createQuery := fmt.Sprintf("drop database %s;", h.dbName)
	_, err = db.Exec(createQuery)
	if err != nil {
		panic(err)
	}

	utils.PanicOnError(db.Close, "closing initial db connection in test")
}

func (h *harness) ApiUrl() string {
	return h.serverUrl
}

func newE2eTestHarness(tokenSourceGenerator func(ctx context.Context, url string) oauth2.TokenSource) *harness {
	ctx := context.Background()
	dc := ioc.NewDependencyCollection()
	clockService, timeSetter := clock.NewMockClock(time.Now())

	sqlbuilder.DefaultFlavor = sqlbuilder.PostgreSQL

	dbName := strings.ReplaceAll("keyline_test_"+uuid.New().String(), "-", "")
	c := config.DatabaseConfig{
		Mode: config.DatabaseModePostgres,
		Postgres: config.PostgresConfig{
			Database: "postgres",
			Host:     "localhost",
			Port:     5732,
			Username: "user",
			Password: "password",
			SslMode:  "disable",
		},
	}

	initDb, err := postgres.ConnectToDatabase(c.Postgres)
	if err != nil {
		panic(err)
	}

	createQuery := fmt.Sprintf("create database %s;", dbName)
	_, err = initDb.Exec(createQuery)
	if err != nil {
		panic(err)
	}
	utils.PanicOnError(initDb.Close, "closing initial db connection in test")

	c.Postgres.Database = dbName
	db, err := setup.Database(dc, c)
	if err != nil {
		panic(fmt.Errorf("failed to create test database: %w", err))
	}

	err = db.Migrate(ctx)
	if err != nil {
		panic(fmt.Errorf("failed to create test database: %w", err))
	}

	ioc.RegisterSingleton(dc, func(dp *ioc.DependencyProvider) clock.Service {
		return clockService
	})
	setup.OutboxDelivery(dc, config.QueueModeNoop)

	vaultPath := fmt.Sprintf("%s/", uuid.New().String())
	setup.KeyServices(dc, config.KeyStoreConfig{
		Mode: config.KeyStoreModeVault,
		Vault: config.VaultKeyStoreConfig{
			Address: "http://localhost:8222",
			Token:   "myroot",
			Mount:   "secret",
			Prefix:  vaultPath,
		},
	})

	setup.Caching(dc, config.CacheModeMemory)
	setup.Services(dc)
	setup.Mediator(dc)

	scope := dc.BuildProvider()

	ctx = middlewares.ContextWithScope(ctx, scope)
	ctx = authentication.ContextWithCurrentUser(ctx, authentication.SystemUser())

	port := findPort()
	serverConfig := config.ServerConfig{
		Port:           port,
		Host:           "localhost",
		AllowedOrigins: []string{"*"},
		ExternalUrl:    fmt.Sprintf("http://localhost:%d", port),
	}
	server.Serve(scope, serverConfig)

	var opts []client.TransportOptions
	if tokenSourceGenerator != nil {
		opts = append(opts, client.WithOidc(tokenSourceGenerator(ctx, serverConfig.ExternalUrl)))
	}

	cl := client.NewClient(serverConfig.ExternalUrl, "test-vs", opts...)

	err = initTest(scope)
	if err != nil {
		panic(fmt.Errorf("failed to initialize test: %w", err))
	}

	return &harness{
		c:         cl,
		scope:     scope,
		ctx:       ctx,
		setTime:   timeSetter,
		dbName:    dbName,
		serverUrl: serverConfig.ExternalUrl,
	}
}

func initTest(dp *ioc.DependencyProvider) error {
	scope := dp.NewScope()
	dbContext := ioc.GetDependency[database.Context](scope)

	ctx := middlewares.ContextWithScope(context.Background(), scope)
	ctx = authentication.ContextWithCurrentUser(ctx, authentication.SystemUser())
	m := ioc.GetDependency[mediatr.Mediator](scope)

	createVirtualServerResponse, err := mediatr.Send[*commands.CreateVirtualServerResponse](ctx, m, commands.CreateVirtualServer{
		Name:               "test-vs",
		DisplayName:        "Test Virtual Server",
		EnableRegistration: true,
		SigningAlgorithm:   config.SigningAlgorithmEdDSA,
	})
	if err != nil {
		return fmt.Errorf("failed to create initial virtual server: %v", err)
	}
	err = dbContext.SaveChanges(ctx)
	if err != nil {
		return fmt.Errorf("failed to save changes to database: %v", err)
	}

	initialAdminUserInfo, err := mediatr.Send[*commands.CreateUserResponse](ctx, m, commands.CreateUser{
		VirtualServerName: "test-vs",
		DisplayName:       "Test Admin User",
		Username:          "test-admin-user",
		Email:             "test-admin-user@localhost",
		EmailVerified:     true,
	})
	if err != nil {
		return fmt.Errorf("failed to create initial admin user: %v", err)
	}
	err = dbContext.SaveChanges(ctx)
	if err != nil {
		return fmt.Errorf("failed to save changes to database: %v", err)
	}

	initialAdminCredential := repositories.NewCredential(initialAdminUserInfo.Id, &repositories.CredentialPasswordDetails{
		HashedPassword: config.C.InitialVirtualServer.Admin.PasswordHash,
		Temporary:      false,
	})
	dbContext.Credentials().Insert(initialAdminCredential)
	err = dbContext.SaveChanges(ctx)
	if err != nil {
		return fmt.Errorf("failed to save changes to database: %v", err)
	}

	_, err = mediatr.Send[*commands.AssignRoleToUserResponse](ctx, m, commands.AssignRoleToUser{
		VirtualServerName: "test-vs",
		ProjectSlug:       createVirtualServerResponse.SystemProjectSlug,
		UserId:            initialAdminUserInfo.Id,
		RoleId:            createVirtualServerResponse.AdminRoleId,
	})
	if err != nil {
		return fmt.Errorf("failed to assign admin role to initial admin user: %v", err)
	}
	err = dbContext.SaveChanges(ctx)
	if err != nil {
		return fmt.Errorf("failed to save changes to database: %v", err)
	}

	serviceUserResponse, err := mediatr.Send[*commands.CreateServiceUserResponse](ctx, m, commands.CreateServiceUser{
		VirtualServerName: "test-vs",
		Username:          serviceUserUsername,
	})
	if err != nil {
		return fmt.Errorf("failed to create initial service user: %v", err)
	}
	err = dbContext.SaveChanges(ctx)
	if err != nil {
		return fmt.Errorf("failed to save changes to database: %v", err)
	}

	_, err = mediatr.Send[*commands.AssignRoleToUserResponse](ctx, m, commands.AssignRoleToUser{
		VirtualServerName: "test-vs",
		ProjectSlug:       createVirtualServerResponse.SystemProjectSlug,
		UserId:            serviceUserResponse.Id,
		RoleId:            createVirtualServerResponse.AdminRoleId,
	})
	if err != nil {
		return fmt.Errorf("failed to assign admin role to service user: %v", err)
	}
	err = dbContext.SaveChanges(ctx)
	if err != nil {
		return fmt.Errorf("failed to save changes to database: %v", err)
	}

	_, err = mediatr.Send[*commands.AssociateServiceUserPublicKeyResponse](ctx, m, commands.AssociateServiceUserPublicKey{
		VirtualServerName: "test-vs",
		ServiceUserId:     serviceUserResponse.Id,
		PublicKey:         serviceUserPublicKey,
		Kid:               utils.Ptr(serviceUserKid),
	})
	if err != nil {
		return fmt.Errorf("failed to associate service user public key: %v", err)
	}
	err = dbContext.SaveChanges(ctx)
	if err != nil {
		return fmt.Errorf("failed to save changes to database: %v", err)
	}

	return scope.Close()
}

var nextPort = 25000
var portMutex = &sync.Mutex{}

func findPort() int {
	portMutex.Lock()
	defer portMutex.Unlock()
	nextPort++
	return nextPort
}
