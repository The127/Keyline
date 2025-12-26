package postgres

import (
	"Keyline/internal/change"
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

type PasswordRuleRepository struct {
	db            *sql.DB
	changeTracker *change.Tracker
	entityType    int
}

func NewPasswordRuleRepository(db *sql.DB, changeTracker change.Tracker, entityType int) repositories.PasswordRuleRepository {
	return &PasswordRuleRepository{
		db:            db,
		changeTracker: &changeTracker,
		entityType:    entityType,
	}
}

func (r *PasswordRuleRepository) selectQuery(filter repositories.PasswordRuleFilter) *sqlbuilder.SelectBuilder {
	s := sqlbuilder.Select(
		"id",
		"audit_created_at",
		"audit_updated_at",
		"version",
		"virtual_server_id",
		"type",
		"details",
	).From("password_rules")

	if filter.HasVirtualServerId() {
		s.Where(s.Equal("virtual_server_id", filter.GetVirtualServerId()))
	}

	if filter.HasType() {
		s.Where(s.Equal("type", filter.GetType()))
	}

	return s
}

func (r *PasswordRuleRepository) List(ctx context.Context, filter repositories.PasswordRuleFilter) ([]*repositories.PasswordRule, error) {
	scope := middlewares.GetScope(ctx)
	dbService := ioc.GetDependency[database.DbService](scope)

	tx, err := dbService.GetTx()
	if err != nil {
		return nil, fmt.Errorf("failed to open tx: %w", err)
	}

	s := r.selectQuery(filter)

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	rows, err := tx.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying db: %w", err)
	}
	defer utils.PanicOnError(rows.Close, "closing rows")

	var result []*repositories.PasswordRule
	for rows.Next() {
		passwordRule := repositories.PasswordRule{
			BaseModel: repositories.NewModelBase(),
		}
		err = rows.Scan(passwordRule.GetScanPointers()...)
		if err != nil {
			return nil, fmt.Errorf("scanning row: %w", err)
		}
		result = append(result, &passwordRule)
	}

	return result, nil
}

func (r *PasswordRuleRepository) Insert(ctx context.Context, passwordRule *repositories.PasswordRule) error {
	scope := middlewares.GetScope(ctx)
	dbService := ioc.GetDependency[database.DbService](scope)

	tx, err := dbService.GetTx()
	if err != nil {
		return fmt.Errorf("failed to open tx: %w", err)
	}

	s := sqlbuilder.InsertInto("password_rules").
		Cols("virtual_server_id", "type", "details").
		Values(
			passwordRule.VirtualServerId(),
			passwordRule.Type(),
			passwordRule.Details(),
		).
		Returning("id", "audit_created_at", "audit_updated_at", "version")

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	row := tx.QueryRowContext(ctx, query, args...)

	err = row.Scan(passwordRule.InsertPointers()...)
	if err != nil {
		return fmt.Errorf("scanning row: %w", err)
	}

	return nil
}

func (r *PasswordRuleRepository) Update(ctx context.Context, passwordRule *repositories.PasswordRule) error {
	scope := middlewares.GetScope(ctx)
	dbService := ioc.GetDependency[database.DbService](scope)

	tx, err := dbService.GetTx()
	if err != nil {
		return fmt.Errorf("failed to open tx: %w", err)
	}

	s := sqlbuilder.Update("password_rules")
	for fieldName, value := range passwordRule.Changes() {
		s.SetMore(s.Assign(fieldName, value))
	}
	s.SetMore(s.Assign("version", passwordRule.Version()+1))

	s.Where(s.Equal("id", passwordRule.Id()))
	s.Where(s.Equal("version", passwordRule.Version()))
	s.Returning("audit_updated_at", "version")

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	row := tx.QueryRowContext(ctx, query, args...)

	err = row.Scan(passwordRule.UpdatePointers()...)
	if err != nil {
		return fmt.Errorf("scanning row: %w", err)
	}

	return nil
}

func (r *PasswordRuleRepository) Delete(ctx context.Context, id uuid.UUID) error {
	scope := middlewares.GetScope(ctx)
	dbService := ioc.GetDependency[database.DbService](scope)

	tx, err := dbService.GetTx()
	if err != nil {
		return fmt.Errorf("failed to open tx: %w", err)
	}

	s := sqlbuilder.DeleteFrom("password_rules")
	s.Where(s.Equal("id", id))

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	_, err = tx.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("executing delete: %w", err)
	}

	return nil
}

func (r *PasswordRuleRepository) First(ctx context.Context, filter repositories.PasswordRuleFilter) (*repositories.PasswordRule, error) {
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

	passwordRule := repositories.PasswordRule{
		BaseModel: repositories.NewModelBase(),
	}
	err = row.Scan(passwordRule.GetScanPointers()...)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, nil
	case err != nil:
		return nil, fmt.Errorf("scanning row: %w", err)
	}

	return &passwordRule, nil
}

func (r *PasswordRuleRepository) Single(ctx context.Context, filter repositories.PasswordRuleFilter) (*repositories.PasswordRule, error) {
	rule, err := r.First(ctx, filter)
	if err != nil {
		return nil, err
	}
	if rule == nil {
		return nil, utils.ErrPasswordRuleNotFound
	}
	return rule, nil
}
