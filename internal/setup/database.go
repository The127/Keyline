package setup

import (
	"Keyline/internal/config"
	"Keyline/internal/database"
	"Keyline/internal/database/postgres"
	"Keyline/internal/logging"
	"context"
	"fmt"

	"github.com/The127/ioc"
)

func Database(dc *ioc.DependencyCollection, c config.DatabaseConfig) (database.Database, error) {
	var db database.Database
	var err error

	switch c.Mode {
	case config.DatabaseModeSqlite:
		panic("not implemented")

	case config.DatabaseModePostgres:
		db, err = postgres.NewPostgresDatabase(c.Postgres)
		if err != nil {
			return nil, fmt.Errorf("connecting to postgres: %w", err)
		}

	default:
		panic("database mode missing or not supported")
	}

	ioc.RegisterSingleton(dc, func(dp *ioc.DependencyProvider) database.Database {
		return db
	})

	ioc.RegisterSingleton(dc, func(dp *ioc.DependencyProvider) database.Factory {
		return database.NewDbFactory(db)
	})

	ioc.RegisterScoped(dc, func(dp *ioc.DependencyProvider) database.Context {
		dbFactory := ioc.GetDependency[database.Factory](dp)
		dbContext, err := dbFactory.NewContext(context.TODO())
		if err != nil {
			logging.Logger.Fatalf("creating database context: %v", err)
		}
		return dbContext
	})
	ioc.RegisterCloseHandler(dc, func(dbService database.Context) error {
		return dbService.SaveChanges(context.TODO())
	})

	return db, nil
}
