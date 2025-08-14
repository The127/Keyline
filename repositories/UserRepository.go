package repositories

import (
	"Keyline/ioc"
	"Keyline/middlewares"
	"Keyline/services"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"time"
)

type User struct {
	id uuid.UUID

	auditCreatedAt time.Time
	auditUpdatedAt time.Time

	virtualServerId uuid.UUID

	username    string
	displayName string
}

type UserFilter struct {
	virtualServerId *uuid.UUID
	username        *string
}

func NewUserFilter() UserFilter {
	return UserFilter{}
}

func (f UserFilter) Clone() UserFilter {
	return f
}

func (f UserFilter) VirtualServer(virtualServerId uuid.UUID) UserFilter {
	filter := f.Clone()
	filter.virtualServerId = &virtualServerId
	return filter
}

func (f UserFilter) Username(username string) UserFilter {
	filter := f.Clone()
	filter.username = &username
	return filter
}

type UserRepository struct {
}

func (r *UserRepository) First(ctx context.Context, filter UserFilter) (*User, error) {
	scope := middlewares.GetScope(ctx)
	dbService := ioc.GetDependency[*services.DbService](scope)

	tx, err := dbService.GetTx()
	if err != nil {
		return nil, fmt.Errorf("failed to open tx: %w", err)
	}

	s := "select id, audit_created_at, audit_updated_at, virtual_server_id, display_name, username from users "
	params := make([]any, 0)

	if filter.username != nil {
		s += fmt.Sprintf(" where username = $%d", len(params)+1)
		params = append(params, filter.username)
	}

	if filter.virtualServerId != nil {
		s += fmt.Sprintf(" where virtual_server_id = $%d", len(params)+1)
		params = append(params, filter.username)
	}

	s += " limit 1"

	row := tx.QueryRow(s, params...)

	var user User
	err = row.Scan(&user.id, &user.auditCreatedAt, &user.auditUpdatedAt, &user.virtualServerId, &user.displayName, &user.username)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, nil

	case err != nil:
		return nil, fmt.Errorf("scanning row: %w", err)
	}

	return &user, nil
}
