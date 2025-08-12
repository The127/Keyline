package commands

import (
	"Keyline/ioc"
	"Keyline/middlewares"
	"Keyline/services"
	"Keyline/utils"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/google/uuid"
)

type RegisterUser struct {
	VirtualServerName string
	DisplayName       string
}

type RegisterUserResponse struct {
	Id uuid.UUID
}

func HandleRegisterUser(ctx context.Context, command RegisterUser) (*RegisterUserResponse, error) {
	scope := middlewares.GetScope(ctx)
	dbService := ioc.GetDependency[*services.DbService](scope)

	tx, err := dbService.GetTx()
	if err != nil {
		return nil, fmt.Errorf("failed to open tx: %w", err)
	}

	// get the virtual server
	row := tx.QueryRow("select id, enable_registration from virtual_servers where name = $1",
		command.VirtualServerName)

	var virtualServerId uuid.UUID
	var isRegistrationEnabled bool
	err = row.Scan(&virtualServerId, &isRegistrationEnabled)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, utils.VirtualServerNotFoundErr
	case err != nil:
		return nil, fmt.Errorf("querying virtual server: %w", err)
	}

	if !isRegistrationEnabled {
		return nil, utils.RegistrationNotEnabledErr
	}

	// create user
	row = tx.QueryRow(`
insert into users
(virtual_server_id, display_name)
values ($1, $2)
returning id;
`, virtualServerId, command.DisplayName)

	var userId uuid.UUID
	err = row.Scan(&userId)
	if err != nil {
		return nil, fmt.Errorf("inserting user: %w", err)
	}

	return &RegisterUserResponse{
		Id: userId,
	}, nil
}
