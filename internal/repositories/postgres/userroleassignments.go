package repositories

import (
	"Keyline/internal/database"
	"Keyline/internal/logging"
	"Keyline/internal/middlewares"
	"Keyline/internal/repositories"
	"Keyline/ioc"
	"Keyline/utils"
	"context"
	"fmt"

	"github.com/huandu/go-sqlbuilder"
)

type userRoleAssignmentRepository struct {
}

func NewUserRoleAssignmentRepository() repositories.UserRoleAssignmentRepository {
	return &userRoleAssignmentRepository{}
}

func (r *userRoleAssignmentRepository) selectQuery(filter repositories.UserRoleAssignmentFilter) *sqlbuilder.SelectBuilder {
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

	if filter.HasUserId() {
		s.Where(s.Equal("ura.user_id", filter.GetUserId()))
	}

	if filter.HasRoleId() {
		s.Where(s.Equal("ura.role_id", filter.GetRoleId()))
	}

	if filter.HasGroupId() {
		s.Where(s.Equal("ura.group_id", filter.GetGroupId()))
	}

	if filter.HasApplicationId() {
		s.Where(s.Equal("ura.application_id", filter.GetApplicationId()))
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
		)
	}

	return s
}

func (r *userRoleAssignmentRepository) List(ctx context.Context, filter repositories.UserRoleAssignmentFilter) ([]*repositories.UserRoleAssignment, int, error) {
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

	var userRoleAssignments []*repositories.UserRoleAssignment
	var totalCount int
	for rows.Next() {
		userRoleAssignment := repositories.UserRoleAssignment{
			ModelBase: repositories.NewModelBase(),
		}
		err = rows.Scan(append(userRoleAssignment.GetScanPointers(filter), &totalCount)...)
		if err != nil {
			return nil, 0, fmt.Errorf("scanning row: %w", err)
		}

		userRoleAssignments = append(userRoleAssignments, &userRoleAssignment)
	}

	return userRoleAssignments, totalCount, nil
}

func (r *userRoleAssignmentRepository) Insert(ctx context.Context, userRoleAssignment *repositories.UserRoleAssignment) error {
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
			userRoleAssignment.UserId(),
			userRoleAssignment.RoleId(),
			userRoleAssignment.GroupId(),
			userRoleAssignment.ApplicationId(),
		).Returning("id", "audit_created_at", "audit_updated_at", "version")

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	row := tx.QueryRowContext(ctx, query, args...)

	err = row.Scan(userRoleAssignment.InsertPointers()...)
	if err != nil {
		return fmt.Errorf("scanning row: %w", err)
	}

	userRoleAssignment.ClearChanges()
	return nil
}
