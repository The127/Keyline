package postgres

import (
	"Keyline/internal/database"
	"Keyline/internal/logging"
	"Keyline/internal/middlewares"
	"Keyline/internal/repositories"
	"Keyline/utils"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/The127/ioc"

	"github.com/google/uuid"
	"github.com/huandu/go-sqlbuilder"
)

type userRepository struct {
}

func NewUserRepository() repositories.UserRepository {
	return &userRepository{}
}

func (r *userRepository) selectQuery(filter repositories.UserFilter) *sqlbuilder.SelectBuilder {
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
		"service_user",
		"metadata",
	).From("users")

	if filter.GetIncludeMetadata() {
		s.SelectMore("metadata")
	}

	if filter.HasUsername() {
		s.Where(s.Equal("username", filter.GetUsername()))
	}

	if filter.HasVirtualServerId() {
		s.Where(s.Equal("virtual_server_id", filter.GetVirtualServerId()))
	}

	if filter.HasId() {
		s.Where(s.Equal("id", filter.GetId()))
	}

	if filter.HasServiceUser() {
		s.Where(s.Equal("service_user", filter.GetServiceUser()))
	}

	if filter.HasSearch() {
		term := filter.GetSearch().Term()
		s.Where(s.Or(
			s.ILike("username", term),
			s.ILike("display_name", term),
		))
	}

	if filter.HasOrder() {
		filter.GetOrderInfo().Apply(s)
	}

	if filter.HasPagination() {
		filter.GetPagingInfo().Apply(s)
	}

	return s
}

func (r *userRepository) List(ctx context.Context, filter repositories.UserFilter) ([]*repositories.User, int, error) {
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

	var users []*repositories.User
	var totalCount int
	for rows.Next() {
		user := repositories.User{
			BaseModel: repositories.NewModelBase(),
		}

		err = rows.Scan(append(user.GetScanPointers(filter), &totalCount)...)
		if err != nil {
			return nil, 0, fmt.Errorf("scanning row: %w", err)
		}

		users = append(users, &user)
	}

	return users, totalCount, nil
}

func (r *userRepository) Single(ctx context.Context, filter repositories.UserFilter) (*repositories.User, error) {
	result, err := r.First(ctx, filter)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, utils.ErrUserNotFound
	}
	return result, nil
}

func (r *userRepository) First(ctx context.Context, filter repositories.UserFilter) (*repositories.User, error) {
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

	user := repositories.User{
		BaseModel: repositories.NewModelBase(),
	}
	err = row.Scan(user.GetScanPointers(filter)...)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, nil

	case err != nil:
		return nil, fmt.Errorf("scanning row: %w", err)
	}

	return &user, nil
}

func (r *userRepository) Update(ctx context.Context, user *repositories.User) error {
	scope := middlewares.GetScope(ctx)
	dbService := ioc.GetDependency[database.DbService](scope)

	tx, err := dbService.GetTx()
	if err != nil {
		return fmt.Errorf("failed to open tx: %w", err)
	}

	s := sqlbuilder.Update("users")
	for fieldName, value := range user.Changes() {
		s.SetMore(s.Assign(fieldName, value))
	}
	s.SetMore(s.Assign("version", user.Version()+1))

	s.Where(s.Equal("id", user.Id()))
	s.Where(s.Equal("version", user.Version()))
	s.Returning("audit_updated_at", "version")

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	row := tx.QueryRowContext(ctx, query, args...)

	err = row.Scan(user.UpdatePointers()...)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return fmt.Errorf("updating user: %w", repositories.ErrVersionMismatch)
	case err != nil:
		return fmt.Errorf("scanning row: %w", err)
	}

	user.ClearChanges()
	return nil
}

func (r *userRepository) Insert(ctx context.Context, user *repositories.User) error {
	scope := middlewares.GetScope(ctx)
	dbService := ioc.GetDependency[database.DbService](scope)

	tx, err := dbService.GetTx()
	if err != nil {
		return fmt.Errorf("failed to open tx: %w", err)
	}

	cols := []string{
		"username",
		"display_name",
		"primary_email",
		"email_verified",
		"service_user",
		"metadata",
	}
	if user.VirtualServerId() != uuid.Nil {
		cols = append(cols, "virtual_server_id")
	} else {
		cols = append(cols, "id")
	}

	s := sqlbuilder.InsertInto("users").
		Cols(cols...)

	values := []any{
		user.Username(),
		user.DisplayName(),
		user.PrimaryEmail(),
		user.EmailVerified(),
		user.IsServiceUser(),
		user.Metadata(),
	}
	if user.VirtualServerId() != uuid.Nil {
		values = append(values, user.VirtualServerId())
	} else {
		values = append(values, user.Id())
	}

	s.Values(values...)

	s.Returning("id", "audit_created_at", "audit_updated_at", "version")

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	row := tx.QueryRowContext(ctx, query, args...)

	err = row.Scan(user.InsertPointers()...)
	if err != nil {
		return fmt.Errorf("scanning row: %w", err)
	}

	user.ClearChanges()
	return nil
}
