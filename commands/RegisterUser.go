package commands

import (
	"Keyline/ioc"
	"Keyline/middlewares"
	"Keyline/repositories"
	"Keyline/services"
	"Keyline/utils"
	"context"
	"fmt"
	"github.com/google/uuid"
)

type RegisterUser struct {
	VirtualServerName string
	DisplayName       string
	Username          string
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

	virtualServerRepository := ioc.GetDependency[*repositories.VirtualServerRepository](scope)
	virtualServer, err := virtualServerRepository.First(ctx, repositories.NewVirtualServerFilter().Name(command.VirtualServerName))
	if err != nil {
		return nil, fmt.Errorf("getting virtual server: %w", err)
	}

	if !virtualServer.EnableRegistration() {
		return nil, utils.ErrRegistrationNotEnabled
	}

	// create user
	row := tx.QueryRow(`
insert into users
(virtual_server_id, display_name, username)
values ($1, $2, $3)
returning id;
`, virtualServer.Id(), command.DisplayName, command.Username)

	var userId uuid.UUID
	err = row.Scan(&userId)
	if err != nil {
		return nil, fmt.Errorf("inserting user: %w", err)
	}

	return &RegisterUserResponse{
		Id: userId,
	}, nil
}
