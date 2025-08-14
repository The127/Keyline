package repositories

import (
	"Keyline/ioc"
	"Keyline/logging"
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

	primaryEmail  string
	emailVerified bool
}

func NewUser(username string, displayName string, primaryEmail string, virtualServerId uuid.UUID) *User {
	return &User{
		virtualServerId: virtualServerId,
		username:        username,
		displayName:     displayName,
		primaryEmail:    primaryEmail,
	}
}

func (m *User) Id() uuid.UUID {
	return m.id
}

func (m *User) AuditCreatedAt() time.Time {
	return m.auditCreatedAt
}

func (m *User) AuditUpdatedAt() time.Time {
	return m.auditUpdatedAt
}

func (m *User) VirtualServerId() uuid.UUID {
	return m.virtualServerId
}

func (m *User) Username() string {
	return m.username
}

func (m *User) DisplayName() string {
	return m.displayName
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

func (r *UserRepository) List(ctx context.Context, filter UserFilter) ([]User, error) {
	scope := middlewares.GetScope(ctx)
	dbService := ioc.GetDependency[*services.DbService](scope)

	tx, err := dbService.GetTx()
	if err != nil {
		return nil, fmt.Errorf("failed to open tx: %w", err)
	}

	s := "select id, audit_created_at, audit_updated_at, virtual_server_id, display_name, username, primary_email, email_verified from users "
	params := make([]any, 0)

	if filter.username != nil {
		s += fmt.Sprintf(" where username = $%d", len(params)+1)
		params = append(params, filter.username)
	}

	if filter.virtualServerId != nil {
		s += fmt.Sprintf(" where virtual_server_id = $%d", len(params)+1)
		params = append(params, filter.username)
	}

	rows, err := tx.Query(s, params...)
	defer rows.Close()
	if err != nil {
		return nil, fmt.Errorf("querying db: %w", err)
	}

	var users []User
	for rows.Next() {
		var user User
		err = rows.Scan(
			&user.id,
			&user.auditCreatedAt,
			&user.auditUpdatedAt,
			&user.virtualServerId,
			&user.displayName,
			&user.username,
			&user.primaryEmail,
			&user.emailVerified,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning row: %w", err)
		}
		users = append(users, user)
	}

	return users, nil
}

func (r *UserRepository) First(ctx context.Context, filter UserFilter) (*User, error) {
	scope := middlewares.GetScope(ctx)
	dbService := ioc.GetDependency[*services.DbService](scope)

	tx, err := dbService.GetTx()
	if err != nil {
		return nil, fmt.Errorf("failed to open tx: %w", err)
	}

	s := "select id, audit_created_at, audit_updated_at, virtual_server_id, display_name, username, primary_email, email_verified from users "
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

	logging.Logger.Debug("sql: %s", s)
	row := tx.QueryRow(s, params...)

	var user User
	err = row.Scan(
		&user.id,
		&user.auditCreatedAt,
		&user.auditUpdatedAt,
		&user.virtualServerId,
		&user.displayName,
		&user.username,
		&user.primaryEmail,
		&user.emailVerified,
	)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, nil

	case err != nil:
		return nil, fmt.Errorf("scanning row: %w", err)
	}

	return &user, nil
}

func (r *UserRepository) Insert(ctx context.Context, user *User) error {
	scope := middlewares.GetScope(ctx)
	dbService := ioc.GetDependency[*services.DbService](scope)

	tx, err := dbService.GetTx()
	if err != nil {
		return fmt.Errorf("failed to open tx: %w", err)
	}

	s := `
insert into users 
    (virtual_server_id, username, display_name, primary_email, email_verified) 
values ($1, $2, $3, $4, $5)
returning id, audit_created_at, audit_updated_at`

	logging.Logger.Debug("sql: %s", s)
	row := tx.QueryRow(
		s,
		user.virtualServerId,
		user.username,
		user.displayName,
		user.primaryEmail,
		user.emailVerified,
	)

	err = row.Scan(&user.id, &user.auditCreatedAt, &user.auditUpdatedAt)
	if err != nil {
		return fmt.Errorf("scanning row: %w", err)
	}

	return nil
}
