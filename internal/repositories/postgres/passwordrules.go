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

type postgresPasswordRule struct {
	postgresBaseModel
	virtualServerId uuid.UUID
	type_           string
	details         []byte
}

func mapPasswordRule(rule *repositories.PasswordRule) *postgresPasswordRule {
	return &postgresPasswordRule{
		postgresBaseModel: mapBase(rule.BaseModel),
		virtualServerId:   rule.VirtualServerId(),
		type_:             string(rule.Type()),
		details:           rule.Details(),
	}
}

func (r *postgresPasswordRule) Map() *repositories.PasswordRule {
	return repositories.NewPasswordRuleFromDB(
		r.MapBase(),
		r.virtualServerId,
		repositories.PasswordRuleType(r.type_),
		r.details,
	)
}

func (r *postgresPasswordRule) scan(row pghelpers.Row) error {
	return row.Scan(
		&r.id,
		&r.auditCreatedAt,
		&r.auditUpdatedAt,
		&r.xmin,
		&r.virtualServerId,
		&r.type_,
	)
}

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

func (r *PasswordRuleRepository) selectQuery(filter *repositories.PasswordRuleFilter) *sqlbuilder.SelectBuilder {
	s := sqlbuilder.Select(
		"id",
		"audit_created_at",
		"audit_updated_at",
		"xmin",
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

func (r *PasswordRuleRepository) List(ctx context.Context, filter *repositories.PasswordRuleFilter) ([]*repositories.PasswordRule, error) {
	s := r.selectQuery(filter)

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying db: %w", err)
	}
	defer utils.PanicOnError(rows.Close, "closing rows")

	var result []*repositories.PasswordRule
	for rows.Next() {
		passwordRule := &postgresPasswordRule{}
		err := passwordRule.scan(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning row: %w", err)
		}
		result = append(result, passwordRule.Map())
	}

	return result, nil
}

func (r *PasswordRuleRepository) First(ctx context.Context, filter *repositories.PasswordRuleFilter) (*repositories.PasswordRule, error) {
	s := r.selectQuery(filter)
	s.Limit(1)

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	row := r.db.QueryRowContext(ctx, query, args...)

	passwordRule := &postgresPasswordRule{}
	err := passwordRule.scan(row)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, nil
	case err != nil:
		return nil, fmt.Errorf("scanning row: %w", err)
	}

	return passwordRule.Map(), nil
}

func (r *PasswordRuleRepository) Single(ctx context.Context, filter *repositories.PasswordRuleFilter) (*repositories.PasswordRule, error) {
	rule, err := r.First(ctx, filter)
	if err != nil {
		return nil, err
	}
	if rule == nil {
		return nil, utils.ErrPasswordRuleNotFound
	}
	return rule, nil
}

func (r *PasswordRuleRepository) Insert(passwordRule *repositories.PasswordRule) {
	r.changeTracker.Add(change.NewEntry(change.Added, r.entityType, passwordRule))
}

func (r *PasswordRuleRepository) ExecuteInsert(ctx context.Context, tx *sql.Tx, passwordRule *repositories.PasswordRule) error {
	mapped := mapPasswordRule(passwordRule)

	s := sqlbuilder.InsertInto("password_rules").
		Cols(
			"id",
			"audit_created_at",
			"audit_updated_at",
			"virtual_server_id",
			"type",
			"details",
		).
		Values(
			mapped.id,
			mapped.auditCreatedAt,
			mapped.auditUpdatedAt,
			mapped.virtualServerId,
			mapped.type_,
			mapped.details,
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

	passwordRule.SetVersion(xmin)
	passwordRule.ClearChanges()
	return nil
}

func (r *PasswordRuleRepository) Update(passwordRule *repositories.PasswordRule) {
	r.changeTracker.Add(change.NewEntry(change.Updated, r.entityType, passwordRule))
}

func (r *PasswordRuleRepository) ExecuteUpdate(ctx context.Context, tx *sql.Tx, passwordRule *repositories.PasswordRule) error {
	if !passwordRule.HasChanges() {
		return nil
	}

	mapped := mapPasswordRule(passwordRule)

	s := sqlbuilder.Update("password_rules")
	s.Where(s.Equal("id", mapped.id))
	s.Where(s.Equal("xmin", mapped.xmin))

	for _, field := range passwordRule.GetChanges() {
		switch field {
		case repositories.PasswordRuleChangeDetails:
			s.SetMore(s.Assign("details", mapped.details))

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
	if err != nil {
		return fmt.Errorf("scanning row: %w", err)
	}

	passwordRule.SetVersion(xmin)
	passwordRule.ClearChanges()
	return nil
}

func (r *PasswordRuleRepository) Delete(id uuid.UUID) {
	r.changeTracker.Add(change.NewEntry(change.Deleted, r.entityType, id))
}

func (r *PasswordRuleRepository) ExecuteDelete(ctx context.Context, tx *sql.Tx, id uuid.UUID) error {
	s := sqlbuilder.DeleteFrom("password_rules")
	s.Where(s.Equal("id", id))

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	_, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("executing delete: %w", err)
	}

	return nil
}
