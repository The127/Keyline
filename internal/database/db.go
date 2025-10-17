package database

import (
	"Keyline/internal/config"
	"Keyline/internal/logging"
	"Keyline/utils"
	"database/sql"
	"embed"
	"fmt"

	_ "github.com/lib/pq"
	"github.com/rubenv/sql-migrate"
)

//go:embed migrations/*
var dbMigrations embed.FS

func Migrate() error {
	migrations := migrate.EmbedFileSystemMigrationSource{
		FileSystem: dbMigrations,
		Root:       "migrations",
	}

	db := ConnectToDatabase(config.C.Database.Postgres)
	defer utils.PanicOnError(db.Close, "failed to close database connection")

	logging.Logger.Infof("Applying migrations...")

	n, err := migrate.Exec(db, "postgres", migrations, migrate.Up)
	if err != nil {
		return fmt.Errorf("failed to apply migrations: %v", err)
	}

	logging.Logger.Infof("Applied %d migrations", n)
	return nil
}

func ConnectToDatabase(pc config.PostgresConfig) *sql.DB {
	logging.Logger.Infof("Connecting to database %s via %s:%d",
		pc.Database,
		pc.Host,
		pc.Port)

	connectionString := fmt.Sprintf("host=%s port=%d dbname=%s user=%s password=%s sslmode=%s",
		pc.Host,
		pc.Port,
		pc.Database,
		pc.Username,
		pc.Password,
		pc.SslMode)

	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		logging.Logger.Fatalf("failed to connect to database: %v", err)
	}

	return db
}
