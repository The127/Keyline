package repositories

import (
	"Keyline/internal/database"
	"Keyline/internal/logging"
	"Keyline/internal/middlewares"
	"Keyline/ioc"
	"Keyline/utils"
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/huandu/go-sqlbuilder"
)

type UserRoleAssignment struct {
	ModelBase

	userId        uuid.UUID
	roleId        uuid.UUID
	groupId       *uuid.UUID
	applicationId *uuid.UUID
}

func NewUserRoleAssignment(userId uuid.UUID, roleId uuid.UUID, groupId *uuid.UUID, applicationId *uuid.UUID) *UserRoleAssignment {
	return &UserRoleAssignment{
		ModelBase:     NewModelBase(),
		userId:        userId,
		roleId:        roleId,
		groupId:       groupId,
		applicationId: applicationId,
	}
}

func (u *UserRoleAssignment) UserId() uuid.UUID {
	return u.userId
}

func (u *UserRoleAssignment) RoleId() uuid.UUID {
	return u.roleId
}

func (u *UserRoleAssignment) GroupId() *uuid.UUID {
	return u.groupId
}

func (u *UserRoleAssignment) ApplicationId() *uuid.UUID {
	return u.applicationId
}

func (u *UserRoleAssignment) getScanPointers() []any {
	return []any{
		&u.id,
		&u.auditCreatedAt,
		&u.auditUpdatedAt,
		&u.version,
		&u.userId,
		&u.roleId,
		&u.groupId,
		&u.applicationId,
	}
}

type UserRoleAssignmentFilter struct {
	userId  *uuid.UUID
	roleId  *uuid.UUID
	groupId *uuid.UUID
}

func NewUserRoleAssignmentFilter() UserRoleAssignmentFilter {
	return UserRoleAssignmentFilter{}
}

func (f UserRoleAssignmentFilter) Clone() UserRoleAssignmentFilter {
	return f
}

func (f UserRoleAssignmentFilter) UserId(userId uuid.UUID) UserRoleAssignmentFilter {
	filter := f.Clone()
	filter.userId = &userId
	return filter
}

func (f UserRoleAssignmentFilter) RoleId(roleId uuid.UUID) UserRoleAssignmentFilter {
	filter := f.Clone()
	filter.roleId = &roleId
	return filter
}

func (f UserRoleAssignmentFilter) GroupId(groupId uuid.UUID) UserRoleAssignmentFilter {
	filter := f.Clone()
	filter.groupId = &groupId
	return filter
}

//go:generate mockgen -destination=./mocks/userroleassignment_repository.go -package=mocks Keyline/internal/repositories UserRoleAssignmentRepository
type UserRoleAssignmentRepository interface {
	Insert(ctx context.Context, userRoleAssignment *UserRoleAssignment) error
	List(ctx context.Context, filter UserRoleAssignmentFilter) ([]*UserRoleAssignment, int, error)
}

type userRoleAssignmentRepository struct {
}

func NewUserRoleAssignmentRepository() UserRoleAssignmentRepository {
	return &userRoleAssignmentRepository{}
}

func (r *userRoleAssignmentRepository) selectQuery(filter UserRoleAssignmentFilter) *sqlbuilder.SelectBuilder {
	s := sqlbuilder.Select(
		"id",
		"audit_created_at",
		"audit_updated_at",
		"version",
		"user_id",
		"role_id",
		"group_id",
		"application_id",
	).From("user_role_assignments")

	if filter.userId != nil {
		s.Where(s.Equal("user_id", filter.userId))
	}

	if filter.roleId != nil {
		s.Where(s.Equal("role_id", filter.roleId))
	}

	if filter.groupId != nil {
		s.Where(s.Equal("group_id", filter.groupId))
	}

	return s
}

func (r *userRoleAssignmentRepository) List(ctx context.Context, filter UserRoleAssignmentFilter) ([]*UserRoleAssignment, int, error) {
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

	var userRoleAssignments []*UserRoleAssignment
	var totalCount int
	for rows.Next() {
		userRoleAssignment := UserRoleAssignment{
			ModelBase: NewModelBase(),
		}
		err = rows.Scan(append(userRoleAssignment.getScanPointers(), &totalCount)...)
		if err != nil {
			return nil, 0, fmt.Errorf("scanning row: %w", err)
		}

		userRoleAssignments = append(userRoleAssignments, &userRoleAssignment)
	}

	return userRoleAssignments, totalCount, nil
}

func (r *userRoleAssignmentRepository) Insert(ctx context.Context, userRoleAssignment *UserRoleAssignment) error {
	scope := middlewares.GetScope(ctx)
	dbService := ioc.GetDependency[database.DbService](scope)

	tx, err := dbService.GetTx()
	if err != nil {
		return fmt.Errorf("failed to open tx: %w", err)
	}

	s := sqlbuilder.InsertInto("user_role_assignments").
		Cols(
			"user_id",
			"role_id",
			"group_id",
			"application_id",
		).
		Values(
			userRoleAssignment.userId,
			userRoleAssignment.roleId,
			userRoleAssignment.groupId,
			userRoleAssignment.applicationId,
		).Returning("id", "audit_created_at", "audit_updated_at", "version")

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	row := tx.QueryRowContext(ctx, query, args...)

	err = row.Scan(&userRoleAssignment.id, &userRoleAssignment.auditCreatedAt, &userRoleAssignment.auditUpdatedAt, &userRoleAssignment.version)
	if err != nil {
		return fmt.Errorf("scanning row: %w", err)
	}

	userRoleAssignment.clearChanges()
	return nil
}
