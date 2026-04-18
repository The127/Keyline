package integration

import (
	"github.com/The127/Keyline/internal/config"
	"github.com/The127/Keyline/internal/database/postgres"
)

// testBackend describes a database backend to run the test suite against.
type testBackend struct {
	name   string
	dbMode config.DatabaseMode
}

// postgresBackendAvailable returns true when the test Postgres instance is reachable.
func postgresBackendAvailable() bool {
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
		return false
	}
	_ = db.Close()
	return true
}

// testBackends is the ordered list of backends the suite runs against.
// Postgres is skipped at runtime when the server is unavailable.
var testBackends = []testBackend{
	{name: "postgres", dbMode: config.DatabaseModePostgres},
	{name: "memory", dbMode: config.DatabaseModeMemory},
}
