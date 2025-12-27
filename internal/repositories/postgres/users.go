package postgres

import (
	"Keyline/internal/change"
	"Keyline/internal/logging"
	"Keyline/internal/repositories"
	"Keyline/internal/repositories/postgres/pghelpers"
	"Keyline/utils"
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/huandu/go-sqlbuilder"
)

type postgresUser struct {
	postgresBaseModel
	virtualServerId uuid.UUID
	username        string
	displayName     string
	primaryEmail    string
	emailVerified   bool
	serviceUser     bool
	metadata        string
}

func mapUser(m *repositories.User) *postgresUser {
	return &postgresUser{
		postgresBaseModel: mapBase(m.BaseModel),
		virtualServerId:   m.VirtualServerId(),
		username:          m.Username(),
		displayName:       m.DisplayName(),
		primaryEmail:      m.PrimaryEmail(),
		emailVerified:     m.EmailVerified(),
		serviceUser:       m.IsServiceUser(),
		metadata:          m.Metadata(),
	}
}

func (u *postgresUser) Map() *repositories.User {
	return repositories.NewUserFromDB(
		u.MapBase(),
		u.virtualServerId,
		u.username,
		u.displayName,
		u.primaryEmail,
		u.emailVerified,
		u.serviceUser,
		u.metadata,
	)
}

func (u *postgresUser) scan(row pghelpers.Row, filter *repositories.UserFilter, additionalPtrs ...any) error {
	ptrs := []any{
		&u.id,
		&u.auditCreatedAt,
		&u.auditUpdatedAt,
		&u.xmin,
		&u.virtualServerId,
		&u.displayName,
		&u.username,
		&u.primaryEmail,
		&u.emailVerified,
		&u.serviceUser,
		&u.metadata,
	}

	if filter.GetIncludeMetadata() {
		ptrs = append(ptrs, &u.metadata)
	}

	ptrs = append(ptrs, additionalPtrs...)

	return row.Scan(ptrs...)
}

type UserRepository struct {
	db            *sql.DB
	changeTracker *change.Tracker
	entityType    int
}

func NewUserRepository(db *sql.DB, changeTracker *change.Tracker, entityType int) *UserRepository {
	return &UserRepository{
		db:            db,
		changeTracker: changeTracker,
		entityType:    entityType,
	}
}

func (r *UserRepository) selectQuery(filter *repositories.UserFilter) *sqlbuilder.SelectBuilder {
	s := sqlbuilder.Select(
		"id",
		"audit_created_at",
		"audit_updated_at",
		"xmin",
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

func (r *UserRepository) List(ctx context.Context, filter *repositories.UserFilter) ([]*repositories.User, int, error) {
	s := r.selectQuery(filter)
	s.SelectMore("count(*) over()")

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("querying db: %w", err)
	}
	defer utils.PanicOnError(rows.Close, "closing rows")

	var users []*repositories.User
	var totalCount int
	for rows.Next() {
		user := &postgresUser{}
		err := user.scan(rows, filter, &totalCount)
		if err != nil {
			return nil, 0, fmt.Errorf("scanning row: %w", err)
		}

		users = append(users, user.Map())
	}

	return users, totalCount, nil
}

func (r *UserRepository) FirstOrErr(ctx context.Context, filter *repositories.UserFilter) (*repositories.User, error) {
	result, err := r.FirstOrNil(ctx, filter)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, utils.ErrUserNotFound
	}
	return result, nil
}

func (r *UserRepository) FirstOrNil(ctx context.Context, filter *repositories.UserFilter) (*repositories.User, error) {
	s := r.selectQuery(filter)
	s.Limit(1)

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	row := r.db.QueryRowContext(ctx, query, args...)

	user := &postgresUser{}
	err := user.scan(row, filter)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, nil

	case err != nil:
		return nil, fmt.Errorf("scanning row: %w", err)
	}

	return user.Map(), nil
}

func (r *UserRepository) Insert(user *repositories.User) {
	r.changeTracker.Add(change.NewEntry(change.Added, r.entityType, user))
}

func (r *UserRepository) ExecuteInsert(ctx context.Context, tx *sql.Tx, user *repositories.User) error {
	mapped := mapUser(user)

	cols := []string{
		"id",
		"audit_created_at",
		"audit_updated_at",
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
		mapped.id,
		mapped.auditCreatedAt,
		mapped.auditUpdatedAt,
		mapped.username,
		mapped.displayName,
		mapped.primaryEmail,
		mapped.emailVerified,
		mapped.serviceUser,
		mapped.metadata,
	}

	if user.VirtualServerId() != uuid.Nil {
		values = append(values, mapped.virtualServerId)
	} else {
		values = append(values, mapped.id)
	}

	s.Values(values...)

	s.Returning("xmin")

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	row := tx.QueryRowContext(ctx, query, args...)

	var xmin uint32
	err := row.Scan(&xmin)
	if err != nil {
		return fmt.Errorf("scanning row: %w", err)
	}

	user.SetVersion(xmin)
	user.ClearChanges()
	return nil
}

func (r *UserRepository) Update(user *repositories.User) {
	r.changeTracker.Add(change.NewEntry(change.Updated, r.entityType, user))
}

func (r *UserRepository) ExecuteUpdate(ctx context.Context, tx *sql.Tx, user *repositories.User) error {
	if !user.HasChanges() {
		return nil
	}

	mapped := mapUser(user)

	s := sqlbuilder.Update("users")
	s.Where(s.Equal("id", mapped.id))
	s.Where(s.Equal("xmin", mapped.xmin))

	for _, field := range user.GetChanges() {
		switch field {
		case repositories.UserChangeDisplayName:
			s.SetMore(s.Assign("display_name", mapped.displayName))

		case repositories.UserChangeEmailVerified:
			s.SetMore(s.Assign("email_verified", mapped.emailVerified))

		default:
			return fmt.Errorf("updating field %v is not supported", field)
		}
	}

	s.Returning("xmin")
	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	row := tx.QueryRowContext(ctx, query, args...)

	var xmin uint32
	err := row.Scan(&xmin)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return fmt.Errorf("updating application: %w", repositories.ErrVersionMismatch)
	case err != nil:
		return fmt.Errorf("scanning row: %w", err)
	}

	user.SetVersion(xmin)
	user.ClearChanges()
	return nil
}
