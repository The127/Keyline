package repositories

import (
	"Keyline/internal/database"
	"Keyline/internal/middlewares"
	"Keyline/ioc"
	"Keyline/logging"
	"Keyline/utils"
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

func (m *User) SetDisplayName(displayName string) {
	m.displayName = displayName
	m.TrackChange("display_name", displayName)
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

func (m *User) getScanPointers() []any {
	return []any{
		&m.id,
		&m.auditCreatedAt,
		&m.auditUpdatedAt,
		&m.version,
		&m.virtualServerId,
		&m.displayName,
		&m.username,
		&m.primaryEmail,
		&m.emailVerified,
	}
}

type UserFilter struct {
	pagingInfo
	orderInfo
	virtualServerId *uuid.UUID
	id              *uuid.UUID
	username        *string
	search          *string
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

func (f UserFilter) GetVirtualServerId() *uuid.UUID {
	return f.virtualServerId
}

func (f UserFilter) Id(id uuid.UUID) UserFilter {
	filter := f.Clone()
	filter.id = &id
	return filter
}

func (f UserFilter) GetId() *uuid.UUID {
	return f.id
}

func (f UserFilter) Username(username string) UserFilter {
	filter := f.Clone()
	filter.username = &username
	return filter
}

func (f UserFilter) Pagination(page int, size int) UserFilter {
	filter := f.Clone()
	filter.pagingInfo = pagingInfo{
		page: page,
		size: size,
	}
	return filter
}

func (f UserFilter) Order(by string, direction string) UserFilter {
	filter := f.Clone()
	filter.orderInfo = orderInfo{
		orderBy:  by,
		orderDir: direction,
	}
	return filter
}

func (f UserFilter) Search(search string) UserFilter {
	filter := f.Clone()
	filter.search = utils.NilIfZero(search)
	return filter
}

func (f UserFilter) GetUsername() *string {
	return f.username
}

//go:generate mockgen -destination=./mocks/user_repository.go -package=mocks Keyline/repositories UserRepository
type UserRepository interface {
	List(ctx context.Context, filter UserFilter) ([]*User, int, error)
	Single(ctx context.Context, filter UserFilter) (*User, error)
	First(ctx context.Context, filter UserFilter) (*User, error)
	Update(ctx context.Context, user *User) error
	Insert(ctx context.Context, user *User) error
}

type userRepository struct {
}

func NewUserRepository() UserRepository {
	return &userRepository{}
}

func (r *userRepository) selectQuery(filter UserFilter) *sqlbuilder.SelectBuilder {
	s := sqlbuilder.Select(
		"id",
		"audit_created_at",
		"audit_updated_at",
		"version",
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

	if filter.search != nil {
		term := "%" + *filter.search + "%"
		s.Where(s.Or(
			s.ILike("username", term),
			s.ILike("display_name", term),
		))
	}

	filter.orderInfo.apply(s)
	filter.pagingInfo.apply(s)

	return s
}

func (r *userRepository) List(ctx context.Context, filter UserFilter) ([]*User, int, error) {
	scope := middlewares.GetScope(ctx)
	dbService := ioc.GetDependency[database.DbService](scope)

	tx, err := dbService.GetTx()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to open tx: %w", err)
	}

	s := r.selectQuery(filter)
	s.SelectMore("count(*) over()")

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	rows, err := tx.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("querying db: %w", err)
	}
	defer utils.PanicOnError(rows.Close, "closing rows")

	var users []*User
	var totalCount int
	for rows.Next() {
		user := User{
			ModelBase: NewModelBase(),
		}

		err = rows.Scan(append(user.getScanPointers(), &totalCount)...)
		if err != nil {
			return nil, 0, fmt.Errorf("scanning row: %w", err)
		}

		users = append(users, &user)
	}

	return users, totalCount, nil
}

func (r *userRepository) Single(ctx context.Context, filter UserFilter) (*User, error) {
	result, err := r.First(ctx, filter)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, utils.ErrUserNotFound
	}
	return result, nil
}

func (r *userRepository) First(ctx context.Context, filter UserFilter) (*User, error) {
	scope := middlewares.GetScope(ctx)
	dbService := ioc.GetDependency[database.DbService](scope)

	tx, err := dbService.GetTx()
	if err != nil {
		return nil, fmt.Errorf("failed to open tx: %w", err)
	}

	s := r.selectQuery(filter)
	s.Limit(1)

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	row := tx.QueryRowContext(ctx, query, args...)

	user := User{
		ModelBase: NewModelBase(),
	}
	err = row.Scan(user.getScanPointers()...)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, nil

	case err != nil:
		return nil, fmt.Errorf("scanning row: %w", err)
	}

	return &user, nil
}

func (r *userRepository) Update(ctx context.Context, user *User) error {
	scope := middlewares.GetScope(ctx)
	dbService := ioc.GetDependency[database.DbService](scope)

	tx, err := dbService.GetTx()
	if err != nil {
		return fmt.Errorf("failed to open tx: %w", err)
	}

	s := sqlbuilder.Update("users")
	for fieldName, value := range user.changes {
		s.SetMore(s.Assign(fieldName, value))
	}
	s.SetMore(s.Assign("version", user.version+1))

	s.Where(s.Equal("id", user.id))
	s.Where(s.Equal("version", user.version))
	s.Returning("audit_updated_at", "version")

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	row := tx.QueryRowContext(ctx, query, args...)

	err = row.Scan(&user.auditUpdatedAt, &user.version)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return fmt.Errorf("updating user: %w", ErrVersionMismatch)
	case err != nil:
		return fmt.Errorf("scanning row: %w", err)
	}

	user.clearChanges()
	return nil
}

func (r *userRepository) Insert(ctx context.Context, user *User) error {
	scope := middlewares.GetScope(ctx)
	dbService := ioc.GetDependency[database.DbService](scope)

	tx, err := dbService.GetTx()
	if err != nil {
		return fmt.Errorf("failed to open tx: %w", err)
	}

	s := sqlbuilder.InsertInto("users").
		Cols(
			"virtual_server_id",
			"username",
			"display_name",
			"primary_email",
			"email_verified",
		).
		Values(
			user.virtualServerId,
			user.username,
			user.displayName,
			user.primaryEmail,
			user.emailVerified,
		).Returning("id", "audit_created_at", "audit_updated_at", "version")

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	row := tx.QueryRowContext(ctx, query, args...)

	err = row.Scan(&user.id, &user.auditCreatedAt, &user.auditUpdatedAt, &user.version)
	if err != nil {
		return fmt.Errorf("scanning row: %w", err)
	}

	user.clearChanges()
	return nil
}
