package postgres

import (
	"Keyline/internal/config"
	db "Keyline/internal/database"
	"Keyline/internal/logging"
	"context"
	"database/sql"
	"embed"
	"fmt"

	migrate "github.com/rubenv/sql-migrate"
)

//go:embed migrations/*
var dbMigrations embed.FS

type database struct {
	db *sql.DB
}

func NewPostgresDatabase(pc config.PostgresConfig) (db.Database, error) {
	dbConnection, err := ConnectToDatabase(pc)
	if err != nil {
		return nil, fmt.Errorf("connecting to database: %v", err)
	}

	return &database{
		db: dbConnection,
	}, nil
}

func ConnectToDatabase(pc config.PostgresConfig) (*sql.DB, error) {
	logging.Logger.Infof("Connecting to database %s via %s:%d",
		pc.Database,
		pc.Host,
		pc.Port,
	)

	connectionString := fmt.Sprintf("host=%s port=%d dbname=%s user=%s password=%s sslmode=%s",
		pc.Host,
		pc.Port,
		pc.Database,
		pc.Username,
		pc.Password,
		pc.SslMode,
	)

	dbConnection, err := sql.Open("postgres", connectionString)
	if err != nil {
		return nil, fmt.Errorf("opening database connection: %w", err)
	}

	return dbConnection, nil
}

func (d *database) Migrate(ctx context.Context) error {
	migrations := migrate.EmbedFileSystemMigrationSource{
		FileSystem: dbMigrations,
		Root:       "migrations",
	}

	logging.Logger.Infof("Applying migrations...")

	n, err := migrate.ExecContext(ctx, d.db, "postgres", migrations, migrate.Up)
	if err != nil {
		return fmt.Errorf("failed to apply migrations: %v", err)
	}

	logging.Logger.Infof("Applied %d migrations", n)
	return nil
}

func (d *database) NewDbContext(_ context.Context) (db.Context, error) {
	return newContext(d.db), nil
}
