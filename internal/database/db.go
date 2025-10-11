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

func Migrate() {
	migrations := migrate.EmbedFileSystemMigrationSource{
		FileSystem: dbMigrations,
		Root:       "migrations",
	}

	db := ConnectToDatabase()
	defer utils.PanicOnError(db.Close, "failed to close database connection")

	logging.Logger.Infof("Applying migrations...")

	n, err := migrate.Exec(db, "postgres", migrations, migrate.Up)
	if err != nil {
		logging.Logger.Fatalf("failed to apply migrations: %v", err)
	}

	logging.Logger.Infof("Applied %d migrations", n)
}

func ConnectToDatabase() *sql.DB {
	logging.Logger.Infof("Connecting to database %s via %s:%d",
		config.C.Database.Database,
		config.C.Database.Host,
		config.C.Database.Port)

	connectionString := fmt.Sprintf("host=%s port=%d dbname=%s user=%s password=%s sslmode=%s",
		config.C.Database.Host,
		config.C.Database.Port,
		config.C.Database.Database,
		config.C.Database.Username,
		config.C.Database.Password,
		config.C.Database.SslMode)

	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		logging.Logger.Fatalf("failed to connect to database: %v", err)
	}

	return db
}
