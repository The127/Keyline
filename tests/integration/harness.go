package integration

import (
	"Keyline/internal/authentication"
	"Keyline/internal/commands"
	"Keyline/internal/config"
	db2 "Keyline/internal/database"
	"Keyline/internal/database/postgres"
	"Keyline/internal/logging"
	"Keyline/internal/middlewares"
	"Keyline/internal/setup"
	"Keyline/utils"
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/The127/go-clock"
	"github.com/The127/ioc"
	"github.com/The127/mediatr"

	"github.com/google/uuid"
	"github.com/huandu/go-sqlbuilder"
)

type harness struct {
	m       mediatr.Mediator
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

func (h *harness) SetTime(t time.Time) {
	h.setTime(t)
}

func (h *harness) VirtualServer() string {
	return "test-vs"
}

func (h *harness) Ctx() context.Context {
	return h.ctx
}

func (h *harness) Mediator() mediatr.Mediator {
	return h.m
}

func newIntegrationTestHarness() *harness {
	ctx := context.Background()
	dc := ioc.NewDependencyCollection()
	c, timeSetter := clock.NewMockClock(time.Now())

	sqlbuilder.DefaultFlavor = sqlbuilder.PostgreSQL

	dbName := strings.ReplaceAll("keyline_test_"+uuid.New().String(), "-", "")
	dbc := config.DatabaseConfig{
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

	initDb, err := postgres.ConnectToDatabase(dbc.Postgres)
	if err != nil {
		panic(err)
	}

	createQuery := fmt.Sprintf("create database %s;", dbName)
	_, err = initDb.Exec(createQuery)
	if err != nil {
		panic(err)
	}
	utils.PanicOnError(initDb.Close, "closing initial db connection in test")

	db, err := setup.Database(dc, dbc)
	if err != nil {
		panic(fmt.Errorf("failed to create test database: %w", err))
	}

	dbc.Postgres.Database = dbName
	err = db.Migrate(ctx)
	if err != nil {
		panic(fmt.Errorf("failed to create test database: %w", err))
	}

	ioc.RegisterSingleton(dc, func(dp *ioc.DependencyProvider) clock.Service {
		return c
	})
	setup.OutboxDelivery(dc, config.QueueModeNoop)
	setup.KeyServices(dc, config.KeyStoreConfig{
		Mode: config.KeyStoreModeMemory,
	})
	setup.Caching(dc, config.CacheModeMemory)
	setup.Services(dc)
	setup.Mediator(dc)

	scope := dc.BuildProvider()
	m := ioc.GetDependency[mediatr.Mediator](scope)

	ctx = middlewares.ContextWithScope(ctx, scope)
	ctx = authentication.ContextWithCurrentUser(ctx, authentication.SystemUser())

	_, err = mediatr.Send[*commands.CreateVirtualServerResponse](ctx, m, commands.CreateVirtualServer{
		Name:               "test-vs",
		DisplayName:        "Test Virtual Server",
		EnableRegistration: true,
		SigningAlgorithm:   config.SigningAlgorithmEdDSA,
	})
	if err != nil {
		logging.Logger.Fatalf("failed to create initial virtual server: %v", err)
	}

	dbContext := ioc.GetDependency[db2.Context](scope)
	err = dbContext.SaveChanges(ctx)
	if err != nil {
		panic(err)
	}

	return &harness{
		m:       m,
		scope:   scope,
		ctx:     ctx,
		setTime: timeSetter,
		dbName:  dbName,
	}
}
