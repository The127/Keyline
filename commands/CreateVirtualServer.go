package commands

import (
	"Keyline/ioc"
	"Keyline/middlewares"
	"Keyline/services"
	"context"
	"fmt"
	"github.com/google/uuid"
)

type CreateVirtualServer struct {
	Name        string
	DisplayName string
}

type CreateVirtualServerResponse struct {
	Id uuid.UUID
}

func HandleCreateVirtualServer(ctx context.Context, command CreateVirtualServer) (*CreateVirtualServerResponse, error) {
	scope := middlewares.GetScope(ctx)
	dbService := ioc.GetDependency[*services.DbService](scope)

	tx, err := dbService.GetTx()
	if err != nil {
		return nil, fmt.Errorf("failed to open tx: %w", err)
	}

	row := tx.QueryRow(`
insert into virtual_servers
("name", "display_name")
values($1, $2)
returning id;`,
		command.Name,
		command.DisplayName)

	var id uuid.UUID
	err = row.Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("failed to execute insert: %w", err)
	}

	return &CreateVirtualServerResponse{
		Id: id,
	}, nil
}
