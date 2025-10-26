package e2e

import (
	"Keyline/client"
	"Keyline/internal/authentication"
	"Keyline/internal/clock"
	"Keyline/internal/commands"
	"Keyline/internal/config"
	"Keyline/internal/database"
	"Keyline/internal/middlewares"
	"Keyline/internal/repositories"
	"Keyline/internal/server"
	"Keyline/internal/setup"
	"Keyline/ioc"
	"Keyline/mediator"
	"Keyline/utils"
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/huandu/go-sqlbuilder"
)

type harness struct {
	c       client.Client
	ctx     context.Context
	setTime clock.TimeSetterFn
	dbName  string
	scope   *ioc.DependencyProvider
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
	dbConnection := ioc.GetDependency[*sql.DB](h.scope)
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

	db := database.ConnectToDatabase(pc)
	createQuery := fmt.Sprintf("drop database %s;", h.dbName)
	_, err := db.Exec(createQuery)
	if err != nil {
		panic(err)
	}

	utils.PanicOnError(db.Close, "closing initial db connection in test")
}

func newE2eTestHarness() *harness {
	ctx := context.Background()
	dc := ioc.NewDependencyCollection()
	clockService, timeSetter := clock.NewMockServiceNow()

	sqlbuilder.DefaultFlavor = sqlbuilder.PostgreSQL

	dbName := strings.ReplaceAll("keyline_test_"+uuid.New().String(), "-", "")
	pc := config.PostgresConfig{
		Database: "postgres",
		Host:     "localhost",
		Port:     5732,
		Username: "user",
		Password: "password",
		SslMode:  "disable",
	}

	db := database.ConnectToDatabase(pc)
	createQuery := fmt.Sprintf("create database %s;", dbName)
	_, err := db.Exec(createQuery)
	if err != nil {
		panic(err)
	}
	utils.PanicOnError(db.Close, "closing initial db connection in test")

	pc.Database = dbName
	err = database.Migrate(pc)
	if err != nil {
		panic(fmt.Errorf("failed to create test database: %w", err))
	}

	ioc.RegisterSingleton(dc, func(dp *ioc.DependencyProvider) clock.Service {
		return clockService
	})
	setup.OutboxDelivery(dc, config.QueueModeNoop)
	setup.KeyServices(dc, config.KeyStoreModeMemory)
	setup.Caching(dc, config.CacheModeMemory)
	setup.Services(dc)
	setup.Repositories(dc, config.DatabaseModePostgres, pc)
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

	c := client.NewClient(serverConfig.ExternalUrl, "test-vs")

	err = initTest(scope)
	if err != nil {
		panic(fmt.Errorf("failed to initialize test: %w", err))
	}

	return &harness{
		c:       c,
		scope:   scope,
		ctx:     ctx,
		setTime: timeSetter,
		dbName:  dbName,
	}
}

func initTest(dp *ioc.DependencyProvider) (err error) {
	scope := dp.NewScope()

	ctx := middlewares.ContextWithScope(context.Background(), scope)
	ctx = authentication.ContextWithCurrentUser(ctx, authentication.SystemUser())
	m := ioc.GetDependency[mediator.Mediator](scope)

	createVirtualServerResponse, err := mediator.Send[*commands.CreateVirtualServerResponse](ctx, m, commands.CreateVirtualServer{
		Name:               "test-vs",
		DisplayName:        "Test Virtual Server",
		EnableRegistration: true,
		SigningAlgorithm:   config.SigningAlgorithmEdDSA,
	})
	if err != nil {
		return fmt.Errorf("failed to create initial virtual server: %v", err)
	}

	initialAdminUserInfo, err := mediator.Send[*commands.CreateUserResponse](ctx, m, commands.CreateUser{
		VirtualServerName: "test-vs",
		DisplayName:       "Test Admin User",
		Username:          "test-admin-user",
		Email:             "test-admin-user@localhost",
		EmailVerified:     true,
	})
	if err != nil {
		return fmt.Errorf("failed to create initial admin user: %v", err)
	}

	credentialRepository := ioc.GetDependency[repositories.CredentialRepository](scope)
	initialAdminCredential := repositories.NewCredential(initialAdminUserInfo.Id, &repositories.CredentialPasswordDetails{
		HashedPassword: config.C.InitialVirtualServer.InitialAdmin.PasswordHash,
		Temporary:      false,
	})
	err = credentialRepository.Insert(ctx, initialAdminCredential)
	if err != nil {
		return fmt.Errorf("failed to create initial admin credential: %v", err)
	}

	_, err = mediator.Send[*commands.AssignRoleToUserResponse](ctx, m, commands.AssignRoleToUser{
		VirtualServerName: "test-vs",
		ProjectSlug:       createVirtualServerResponse.SystemProjectSlug,
		UserId:            initialAdminUserInfo.Id,
		RoleId:            createVirtualServerResponse.AdminRoleId,
	})
	if err != nil {
		return fmt.Errorf("failed to assign admin role to initial admin user: %v", err)
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
