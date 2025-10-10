package repositories

import (
	"Keyline/internal/database"
	"Keyline/internal/middlewares"
	"Keyline/ioc"
	"Keyline/logging"
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/huandu/go-sqlbuilder"
)

type UserRoleAssignment struct {
	ModelBase

	userId  uuid.UUID
	roleId  uuid.UUID
	groupId *uuid.UUID
}

func NewUserRoleAssignment(userId uuid.UUID, roleId uuid.UUID, groupId *uuid.UUID) *UserRoleAssignment {
	return &UserRoleAssignment{
		ModelBase: NewModelBase(),
		userId:    userId,
		roleId:    roleId,
		groupId:   groupId,
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

//go:generate mockgen -destination=./mocks/userroleassignment_repository.go -package=mocks Keyline/repositories UserRoleAssignmentRepository
type UserRoleAssignmentRepository interface {
	Insert(ctx context.Context, userRoleAssignment *UserRoleAssignment) error
}

type userRoleAssignmentRepository struct {
}

func NewUserRoleAssignmentRepository() UserRoleAssignmentRepository {
	return &userRoleAssignmentRepository{}
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
		).
		Values(
			userRoleAssignment.userId,
			userRoleAssignment.roleId,
			userRoleAssignment.groupId,
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
