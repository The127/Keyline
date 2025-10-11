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

	userInfo UserRoleAssignmentUserInfo
	roleInfo UserRoleAssignmentRoleInfo
}

type UserRoleAssignmentUserInfo struct {
	Username    string
	DisplayName string
}

type UserRoleAssignmentRoleInfo struct {
	Name string
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

func (u *UserRoleAssignment) UserInfo() UserRoleAssignmentUserInfo {
	return u.userInfo
}

func (u *UserRoleAssignment) RoleInfo() UserRoleAssignmentRoleInfo {
	return u.roleInfo
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

func (u *UserRoleAssignment) getScanPointers(filter UserRoleAssignmentFilter) []any {
	ptrs := []any{
		&u.id,
		&u.auditCreatedAt,
		&u.auditUpdatedAt,
		&u.version,
		&u.userId,
		&u.roleId,
		&u.groupId,
		&u.applicationId,
	}

	if filter.includeUser {
		ptrs = append(ptrs,
			&u.userInfo.Username,
			&u.userInfo.DisplayName,
		)
	}

	if filter.includeRole {
		ptrs = append(ptrs,
			&u.roleInfo.Name,
		)
	}

	return ptrs
}

type UserRoleAssignmentFilter struct {
	userId        *uuid.UUID
	roleId        *uuid.UUID
	groupId       *uuid.UUID
	applicationId *uuid.UUID
	includeUser   bool
	includeRole   bool
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

func (f UserRoleAssignmentFilter) ApplicationId(applicationId uuid.UUID) UserRoleAssignmentFilter {
	filter := f.Clone()
	filter.applicationId = &applicationId
	return filter
}

func (f UserRoleAssignmentFilter) IncludeUser() UserRoleAssignmentFilter {
	filter := f.Clone()
	filter.includeUser = true
	return filter
}

func (f UserRoleAssignmentFilter) IncludeRole() UserRoleAssignmentFilter {
	filter := f.Clone()
	filter.includeRole = true
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
		"ura.id",
		"ura.audit_created_at",
		"ura.audit_updated_at",
		"ura.version",
		"ura.user_id",
		"ura.role_id",
		"ura.group_id",
		"ura.application_id",
	).From("user_role_assignments as ura")

	if filter.userId != nil {
		s.Where(s.Equal("ura.user_id", filter.userId))
	}

	if filter.roleId != nil {
		s.Where(s.Equal("ura.role_id", filter.roleId))
	}

	if filter.groupId != nil {
		s.Where(s.Equal("ura.group_id", filter.groupId))
	}

	if filter.applicationId != nil {
		s.Where(s.Equal("ura.application_id", filter.applicationId))
	}

	if filter.includeUser {
		s.Join("users as u", "u.id = ura.user_id")
		s.SelectMore(
			"u.username",
			"u.display_name",
		)
	}

	if filter.includeRole {
		s.Join("roles as r", "r.id = ura.role_id")
		s.SelectMore(
			"r.name",
		)
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
		err = rows.Scan(append(userRoleAssignment.getScanPointers(filter), &totalCount)...)
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
