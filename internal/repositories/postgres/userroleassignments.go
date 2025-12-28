package postgres

import (
	"Keyline/internal/change"
	"Keyline/internal/logging"
	"Keyline/internal/repositories"
	"Keyline/internal/repositories/postgres/pghelpers"
	"Keyline/utils"
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"

	"github.com/huandu/go-sqlbuilder"
)

type postgresUserRoleAssignment struct {
	postgresBaseModel
	userId   uuid.UUID
	roleId   uuid.UUID
	groupId  *uuid.UUID
	userInfo *postgresUserRoleAssignmentUserInfo
	roleInfo *postgresUserRoleAssignmentRoleInfo
}

type postgresUserRoleAssignmentUserInfo struct {
	Username    string
	DisplayName string
}

type postgresUserRoleAssignmentRoleInfo struct {
	ProjectSlug string
	Name        string
}

func mapUserRoleAssignment(userRoleAssignment *repositories.UserRoleAssignment) *postgresUserRoleAssignment {
	return &postgresUserRoleAssignment{
		postgresBaseModel: mapBase(userRoleAssignment.BaseModel),
		userId:            userRoleAssignment.UserId(),
		roleId:            userRoleAssignment.RoleId(),
		groupId:           userRoleAssignment.GroupId(),
	}
}

func (a *postgresUserRoleAssignment) Map() *repositories.UserRoleAssignment {
	var userRoleAssignmentUserInfo *repositories.UserRoleAssignmentUserInfo
	if a.userInfo != nil {
		userRoleAssignmentUserInfo = &repositories.UserRoleAssignmentUserInfo{
			Username:    a.userInfo.Username,
			DisplayName: a.userInfo.DisplayName,
		}
	}

	var userRoleAssignmentRoleInfo *repositories.UserRoleAssignmentRoleInfo
	if a.roleInfo != nil {
		userRoleAssignmentRoleInfo = &repositories.UserRoleAssignmentRoleInfo{
			ProjectSlug: a.roleInfo.ProjectSlug,
			Name:        a.roleInfo.Name,
		}
	}

	return repositories.NewUserRoleAssignmentFromDB(
		a.MapBase(),
		a.userId,
		a.roleId,
		a.groupId,
		userRoleAssignmentUserInfo,
		userRoleAssignmentRoleInfo,
	)
}

func (a *postgresUserRoleAssignment) scan(row pghelpers.Row, filter *repositories.UserRoleAssignmentFilter, additionalPtrs ...any) error {
	ptrs := []any{
		&a.id,
		&a.auditCreatedAt,
		&a.auditUpdatedAt,
		&a.xmin,
		&a.userId,
		&a.roleId,
		&a.groupId,
	}

	if filter.GetIncludeUser() {
		a.userInfo = &postgresUserRoleAssignmentUserInfo{}
		ptrs = append(ptrs,
			&a.userInfo.Username,
			&a.userInfo.DisplayName,
		)
	}

	if filter.GetIncludeRole() {
		a.roleInfo = &postgresUserRoleAssignmentRoleInfo{}
		ptrs = append(ptrs,
			&a.roleInfo.Name,
			&a.roleInfo.ProjectSlug,
		)
	}

	ptrs = append(ptrs, additionalPtrs...)

	return row.Scan(ptrs...)
}

type UserRoleAssignmentRepository struct {
	db            *sql.DB
	changeTracker *change.Tracker
	entityType    int
}

func NewUserRoleAssignmentRepository(db *sql.DB, changeTracker *change.Tracker, entityType int) *UserRoleAssignmentRepository {
	return &UserRoleAssignmentRepository{
		db:            db,
		changeTracker: changeTracker,
		entityType:    entityType,
	}
}

func (r *UserRoleAssignmentRepository) selectQuery(filter *repositories.UserRoleAssignmentFilter) *sqlbuilder.SelectBuilder {
	s := sqlbuilder.Select(
		"ura.id",
		"ura.audit_created_at",
		"ura.audit_updated_at",
		"ura.xmin",
		"ura.user_id",
		"ura.role_id",
		"ura.group_id",
	).From("user_role_assignments as ura")

	if filter.HasUserId() {
		s.Where(s.Equal("ura.user_id", filter.GetUserId()))
	}

	if filter.HasRoleId() {
		s.Where(s.Equal("ura.role_id", filter.GetRoleId()))
	}

	if filter.HasGroupId() {
		s.Where(s.Equal("ura.group_id", filter.GetGroupId()))
	}

	if filter.GetIncludeUser() {
		s.Join("users as u", "u.id = ura.user_id")
		s.SelectMore(
			"u.username",
			"u.display_name",
		)
	}

	if filter.GetIncludeRole() {
		s.Join("roles as r", "r.id = ura.role_id")
		s.SelectMore(
			"r.name",
			"(select slug from projects where id = r.project_id) as project_slug",
		)
	}

	return s
}

func (r *UserRoleAssignmentRepository) List(ctx context.Context, filter *repositories.UserRoleAssignmentFilter) ([]*repositories.UserRoleAssignment, int, error) {
	s := r.selectQuery(filter)
	s.SelectMore("count(*) over()")

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("querying db: %w", err)
	}
	defer utils.PanicOnError(rows.Close, "closing rows")

	var userRoleAssignments []*repositories.UserRoleAssignment
	var totalCount int
	for rows.Next() {
		userRoleAssignment := &postgresUserRoleAssignment{}
		err := userRoleAssignment.scan(rows, filter, &totalCount)
		if err != nil {
			return nil, 0, fmt.Errorf("scanning row: %w", err)
		}

		userRoleAssignments = append(userRoleAssignments, userRoleAssignment.Map())
	}

	return userRoleAssignments, totalCount, nil
}

func (r *UserRoleAssignmentRepository) Insert(userRoleAssignment *repositories.UserRoleAssignment) {
	r.changeTracker.Add(change.NewEntry(change.Added, r.entityType, userRoleAssignment))
}

func (r *UserRoleAssignmentRepository) ExecuteInsert(ctx context.Context, tx *sql.Tx, userRoleAssignment *repositories.UserRoleAssignment) error {
	mapped := mapUserRoleAssignment(userRoleAssignment)

	s := sqlbuilder.InsertInto("user_role_assignments").
		Cols(
			"id",
			"audit_created_at",
			"audit_updated_at",
			"user_id",
			"role_id",
			"group_id",
		).
		Values(
			mapped.id,
			mapped.auditCreatedAt,
			mapped.auditUpdatedAt,
			mapped.userId,
			mapped.roleId,
			mapped.groupId,
		).
		Returning("xmin")

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	row := tx.QueryRowContext(ctx, query, args...)

	var xmin uint32
	err := row.Scan(&xmin)
	if err != nil {
		return fmt.Errorf("scanning row: %w", err)
	}

	userRoleAssignment.SetVersion(xmin)
	return nil
}
