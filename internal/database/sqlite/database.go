package sqlite

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"github.com/The127/Keyline/config"
	db "github.com/The127/Keyline/internal/database"
	"github.com/The127/Keyline/internal/logging"

	migrate "github.com/rubenv/sql-migrate"
	_ "modernc.org/sqlite"
)

//go:embed migrations/*
var dbMigrations embed.FS

type database struct {
	db *sql.DB
}

func NewSqliteDatabase(sc config.SqliteConfig) (db.Database, error) {
	dbConnection, err := ConnectToDatabase(sc)
	if err != nil {
		return nil, fmt.Errorf("connecting to database: %w", err)
	}

	return &database{
		db: dbConnection,
	}, nil
}

func ConnectToDatabase(sc config.SqliteConfig) (*sql.DB, error) {
	logging.Logger.Infof("Opening sqlite database %s", sc.Database)

	if dir := filepath.Dir(sc.Database); dir != "." {
		err := os.MkdirAll(dir, 0o750)
		if err != nil {
			return nil, fmt.Errorf("creating sqlite database directory: %w", err)
		}
	}

	dbConnection, err := sql.Open("sqlite", connectionString(sc))
	if err != nil {
		return nil, fmt.Errorf("opening database connection: %w", err)
	}

	dbConnection.SetMaxOpenConns(1)

	err = dbConnection.Ping()
	if err != nil {
		return nil, fmt.Errorf("pinging database: %w", err)
	}

	return dbConnection, nil
}

func connectionString(sc config.SqliteConfig) string {
	query := url.Values{}
	query.Add("_pragma", "foreign_keys(1)")
	query.Add("_pragma", "journal_mode(WAL)")
	query.Add("_pragma", "busy_timeout(10000)")
	query.Set("_txlock", "immediate")
	query.Set("_time_format", "sqlite")
	query.Set("_timezone", "UTC")

	u := url.URL{
		Scheme:   "file",
		Path:     sc.Database,
		RawQuery: query.Encode(),
		OmitHost: true,
	}

	return u.String()
}

func (d *database) Migrate(ctx context.Context) error {
	migrations := migrate.EmbedFileSystemMigrationSource{
		FileSystem: dbMigrations,
		Root:       "migrations",
	}

	logging.Logger.Infof("Applying migrations...")

	n, err := migrate.ExecContext(ctx, d.db, "sqlite3", migrations, migrate.Up)
	if err != nil {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	logging.Logger.Infof("Applied %d migrations", n)
	return nil
}

func (d *database) NewDbContext(_ context.Context) (db.Context, error) {
	return newContext(d.db), nil
}

func (d *database) Close() error {
	return d.db.Close()
}
