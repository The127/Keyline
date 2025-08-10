package virtualServers

import (
	"Keyline/ioc"
	"Keyline/middlewares"
	"Keyline/services"
	"context"
	"fmt"
)

type DoesAnyVirtualServerExistQuery struct{}
type DoesAnyVirtualServerExistResponse struct {
	FoundVirtualServer bool
}

func DoesAnyVirtualServerExistQueryHandler(ctx context.Context, _ DoesAnyVirtualServerExistQuery) (DoesAnyVirtualServerExistResponse, error) {
	scope := middlewares.GetScope(ctx)
	dbService := ioc.GetDependency[*services.DbService](scope)

	tx, err := dbService.GetTx()
	if err != nil {
		return DoesAnyVirtualServerExistResponse{}, fmt.Errorf("failed to open tx: %w", err)
	}

	row := tx.QueryRow("select exists(select * from virtual_servers)")

	var anyVirtualServers bool
	err = row.Scan(&anyVirtualServers)
	if err != nil {
		return DoesAnyVirtualServerExistResponse{}, fmt.Errorf("failed to query db: %w", err)
	}

	return DoesAnyVirtualServerExistResponse{
		FoundVirtualServer: anyVirtualServers,
	}, nil
}
