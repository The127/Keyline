package queries

import (
	"Keyline/ioc"
	"Keyline/middlewares"
	"Keyline/services"
	"context"
	"fmt"
)

type AnyVirtualServerExists struct{}
type AnyVirtualServerExistsResult struct {
	Found bool
}

func HandleAnyVirtualServerExists(ctx context.Context, _ AnyVirtualServerExists) (*AnyVirtualServerExistsResult, error) {
	scope := middlewares.GetScope(ctx)
	dbService := ioc.GetDependency[*services.DbService](scope)

	tx, err := dbService.GetTx()
	if err != nil {
		return nil, fmt.Errorf("failed to open tx: %w", err)
	}

	row := tx.QueryRow("select exists(select * from virtual_servers)")

	var anyVirtualServers bool
	err = row.Scan(&anyVirtualServers)
	if err != nil {
		return nil, fmt.Errorf("failed to query db: %w", err)
	}

	return &AnyVirtualServerExistsResult{
		Found: anyVirtualServers,
	}, nil
}
