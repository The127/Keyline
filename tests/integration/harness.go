package integration

import (
	"Keyline/internal/authentication"
	"Keyline/internal/clock"
	"Keyline/internal/commands"
	"Keyline/internal/config"
	"Keyline/internal/database"
	"Keyline/internal/logging"
	"Keyline/internal/middlewares"
	"Keyline/internal/setup"
	"Keyline/ioc"
	"Keyline/mediator"
	"Keyline/utils"
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/huandu/go-sqlbuilder"
)

type harness struct {
	m       mediator.Mediator
	scope   *ioc.DependencyProvider
	ctx     context.Context
	setTime clock.TimeSetterFn
	dbName  string
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

func (h *harness) SetTime(t time.Time) {
	h.setTime(t)
}

func (h *harness) VirtualServer() string {
	return "test-vs"
}

func (h *harness) Ctx() context.Context {
	return h.ctx
}

func (h *harness) Mediator() mediator.Mediator {
	return h.m
}

func newIntegrationTestHarness() *harness {
	ctx := context.Background()
	dc := ioc.NewDependencyCollection()
	c, timeSetter := clock.NewMockServiceNow()

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
		return c
	})
	setup.KeyServices(dc, config.KeyStoreModeMemory)
	setup.Caching(dc, config.CacheModeMemory)
	setup.Services(dc)
	setup.Repositories(dc, config.DatabaseModePostgres, pc)
	setup.Mediator(dc)

	scope := dc.BuildProvider()
	m := ioc.GetDependency[mediator.Mediator](scope)

	ctx = middlewares.ContextWithScope(ctx, scope)
	ctx = authentication.ContextWithCurrentUser(ctx, authentication.SystemUser())

	_, err = mediator.Send[*commands.CreateVirtualServerResponse](ctx, m, commands.CreateVirtualServer{
		Name:               "test-vs",
		DisplayName:        "Test Virtual Server",
		EnableRegistration: true,
		SigningAlgorithm:   config.SigningAlgorithmEdDSA,
	})
	if err != nil {
		logging.Logger.Fatalf("failed to create initial virtual server: %v", err)
	}

	return &harness{
		m:       m,
		scope:   scope,
		ctx:     ctx,
		setTime: timeSetter,
		dbName:  dbName,
	}
}
