package repositories

import (
	"Keyline/database"
	"Keyline/ioc"
	"Keyline/logging"
	"Keyline/middlewares"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/huandu/go-sqlbuilder"
)

type User struct {
	ModelBase

	virtualServerId uuid.UUID

	username    string
	displayName string

	primaryEmail  string
	emailVerified bool
}

func NewUser(username string, displayName string, primaryEmail string, virtualServerId uuid.UUID) *User {
	return &User{
		ModelBase:       NewModelBase(),
		virtualServerId: virtualServerId,
		username:        username,
		displayName:     displayName,
		primaryEmail:    primaryEmail,
	}
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

func (m *User) PrimaryEmail() string {
	return m.primaryEmail
}

func (m *User) EmailVerified() bool {
	return m.emailVerified
}

func (m *User) SetEmailVerified(emailVerified bool) {
	m.emailVerified = emailVerified
	m.TrackChange("email_verified", emailVerified)
}

type UserFilter struct {
	virtualServerId *uuid.UUID
	id              *uuid.UUID
	username        *string
}

func NewUserFilter() UserFilter {
	return UserFilter{}
}

func (f UserFilter) Clone() UserFilter {
	return f
}

func (f UserFilter) VirtualServerId(virtualServerId uuid.UUID) UserFilter {
	filter := f.Clone()
	filter.virtualServerId = &virtualServerId
	return filter
}

func (f UserFilter) Id(id uuid.UUID) UserFilter {
	filter := f.Clone()
	filter.id = &id
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
	dbService := ioc.GetDependency[*database.DbService](scope)

	tx, err := dbService.GetTx()
	if err != nil {
		return nil, fmt.Errorf("failed to open tx: %w", err)
	}

	s := sqlbuilder.Select("id", "audit_created_at", "audit_updated_at", "virtual_server_id", "display_name", "username", "primary_email", "email_verified")

	if filter.username != nil {
		s.Where(s.Equal("username", filter.username))
	}

	if filter.virtualServerId != nil {
		s.Where(s.Equal("virtual_server_id", filter.virtualServerId))
	}

	if filter.id != nil {
		s.Where(s.Equal("id", filter.id))
	}

	query, args := s.Build()
	logging.Logger.Debug("sql: %s", query)
	rows, err := tx.Query(query, args...)
	defer rows.Close()
	if err != nil {
		return nil, fmt.Errorf("querying db: %w", err)
	}

	var users []User
	for rows.Next() {
		user := User{
			ModelBase: NewModelBase(),
		}
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
	dbService := ioc.GetDependency[*database.DbService](scope)

	tx, err := dbService.GetTx()
	if err != nil {
		return nil, fmt.Errorf("failed to open tx: %w", err)
	}

	s := sqlbuilder.Select(
		"id",
		"audit_created_at",
		"audit_updated_at",
		"virtual_server_id",
		"display_name",
		"username",
		"primary_email",
		"email_verified",
	).From("users")

	if filter.username != nil {
		s.Where(s.Equal("username", filter.username))
	}

	if filter.virtualServerId != nil {
		s.Where(s.Equal("virtual_server_id", filter.virtualServerId))
	}

	if filter.id != nil {
		s.Where(s.Equal("id", filter.id))
	}

	s.Limit(1)

	query, args := s.Build()
	logging.Logger.Debug("sql: %s", query)
	row := tx.QueryRow(query, args...)

	user := User{
		ModelBase: NewModelBase(),
	}
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

func (r *UserRepository) Update(ctx context.Context, user *User) error {
	scope := middlewares.GetScope(ctx)
	dbService := ioc.GetDependency[*database.DbService](scope)

	tx, err := dbService.GetTx()
	if err != nil {
		return fmt.Errorf("failed to open tx: %w", err)
	}

	s := sqlbuilder.Update("users")
	for fieldName, value := range user.changes {
		s.Set(s.Assign(fieldName, value))
	}

	s.Where(s.Equal("id", user.id))
	s.Returning("audit_updated_at")

	query, args := s.Build()
	logging.Logger.Debug("sql: %s", query)
	row := tx.QueryRow(query, args...)

	err = row.Scan(&user.auditUpdatedAt)
	if err != nil {
		return fmt.Errorf("scanning row: %w", err)
	}

	return nil
}

func (r *UserRepository) Insert(ctx context.Context, user *User) error {
	scope := middlewares.GetScope(ctx)
	dbService := ioc.GetDependency[*database.DbService](scope)

	tx, err := dbService.GetTx()
	if err != nil {
		return fmt.Errorf("failed to open tx: %w", err)
	}

	s := sqlbuilder.InsertInto("users").
		Cols("virtual_server_id", "username", "display_name", "primary_email", "email_verified").
		Values(
			user.virtualServerId,
			user.username,
			user.displayName,
			user.primaryEmail,
			user.emailVerified,
		).Returning("id", "audit_created_at", "audit_updated_at")

	query, args := s.Build()
	logging.Logger.Debug("sql: %s", query)
	row := tx.QueryRow(query, args...)

	err = row.Scan(&user.id, &user.auditCreatedAt, &user.auditUpdatedAt)
	if err != nil {
		return fmt.Errorf("scanning row: %w", err)
	}

	return nil
}
