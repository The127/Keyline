package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/The127/Keyline/internal/change"
	"github.com/The127/Keyline/internal/logging"
	"github.com/The127/Keyline/internal/repositories"
	"github.com/The127/Keyline/internal/repositories/postgres/pghelpers"
	"github.com/The127/Keyline/utils"

	"github.com/google/uuid"
	"github.com/huandu/go-sqlbuilder"
)

type postgresApplicationKey struct {
	postgresBaseModel
	applicationId uuid.UUID
	kid           string
	publicKey     string
}

func mapApplicationKey(applicationKey *repositories.ApplicationKey) *postgresApplicationKey {
	return &postgresApplicationKey{
		postgresBaseModel: mapBase(applicationKey.BaseModel),
		applicationId:     applicationKey.ApplicationId(),
		kid:               applicationKey.Kid(),
		publicKey:         applicationKey.PublicKey(),
	}
}

func (k *postgresApplicationKey) Map() *repositories.ApplicationKey {
	return repositories.NewApplicationKeyFromDB(
		k.MapBase(),
		k.applicationId,
		k.kid,
		k.publicKey,
	)
}

func (k *postgresApplicationKey) scan(row pghelpers.Row, additionalPtrs ...any) error {
	ptrs := []any{
		&k.id,
		&k.auditCreatedAt,
		&k.auditUpdatedAt,
		&k.xmin,
		&k.applicationId,
		&k.kid,
		&k.publicKey,
	}

	ptrs = append(ptrs, additionalPtrs...)

	return row.Scan(ptrs...)
}

type ApplicationKeyRepository struct {
	db            *sql.DB
	changeTracker *change.Tracker
	entityType    int
}

func NewApplicationKeyRepository(db *sql.DB, changeTracker *change.Tracker, entityType int) *ApplicationKeyRepository {
	return &ApplicationKeyRepository{
		db:            db,
		changeTracker: changeTracker,
		entityType:    entityType,
	}
}

func (r *ApplicationKeyRepository) selectQuery(filter *repositories.ApplicationKeyFilter) *sqlbuilder.SelectBuilder {
	s := sqlbuilder.Select(
		"id",
		"audit_created_at",
		"audit_updated_at",
		"xmin",
		"application_id",
		"kid",
		"public_key",
	).From("application_keys")

	if filter.HasId() {
		s.Where(s.Equal("id", filter.GetId()))
	}

	if filter.HasApplicationId() {
		s.Where(s.Equal("application_id", filter.GetApplicationId()))
	}

	if filter.HasKid() {
		s.Where(s.Equal("kid", filter.GetKid()))
	}

	s.OrderBy("audit_created_at")

	return s
}

func (r *ApplicationKeyRepository) FirstOrErr(ctx context.Context, filter *repositories.ApplicationKeyFilter) (*repositories.ApplicationKey, error) {
	result, err := r.FirstOrNil(ctx, filter)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, utils.ErrApplicationKeyNotFound
	}
	return result, nil
}

func (r *ApplicationKeyRepository) FirstOrNil(ctx context.Context, filter *repositories.ApplicationKeyFilter) (*repositories.ApplicationKey, error) {
	s := r.selectQuery(filter)
	s.Limit(1)

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	row := r.db.QueryRowContext(ctx, query, args...)

	applicationKey := &postgresApplicationKey{}
	err := applicationKey.scan(row)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, nil

	case err != nil:
		return nil, fmt.Errorf("scanning row: %w", err)
	}

	return applicationKey.Map(), nil
}

func (r *ApplicationKeyRepository) List(ctx context.Context, filter *repositories.ApplicationKeyFilter) ([]*repositories.ApplicationKey, error) {
	s := r.selectQuery(filter)

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying db: %w", err)
	}
	defer utils.PanicOnError(rows.Close, "closing rows")

	var applicationKeys []*repositories.ApplicationKey
	for rows.Next() {
		applicationKey := &postgresApplicationKey{}
		err = applicationKey.scan(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning row: %w", err)
		}

		applicationKeys = append(applicationKeys, applicationKey.Map())
	}

	return applicationKeys, nil
}

func (r *ApplicationKeyRepository) Insert(applicationKey *repositories.ApplicationKey) {
	r.changeTracker.Add(change.NewEntry(change.Added, r.entityType, applicationKey))
}

func (r *ApplicationKeyRepository) ExecuteInsert(ctx context.Context, tx *sql.Tx, applicationKey *repositories.ApplicationKey) error {
	mapped := mapApplicationKey(applicationKey)

	s := sqlbuilder.InsertInto("application_keys").
		Cols(
			"id",
			"audit_created_at",
			"audit_updated_at",
			"application_id",
			"kid",
			"public_key",
		).
		Values(
			mapped.id,
			mapped.auditCreatedAt,
			mapped.auditUpdatedAt,
			mapped.applicationId,
			mapped.kid,
			mapped.publicKey,
		).
		Returning("xmin")

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	row := tx.QueryRowContext(ctx, query, args...)

	var xmin uint32
	err := row.Scan(&xmin)
	if err != nil {
		return fmt.Errorf("inserting application key: %w", err)
	}

	applicationKey.SetVersion(xmin)
	applicationKey.ClearChanges()
	return nil
}

func (r *ApplicationKeyRepository) Delete(id uuid.UUID) {
	r.changeTracker.Add(change.NewEntry(change.Deleted, r.entityType, id))
}

func (r *ApplicationKeyRepository) ExecuteDelete(ctx context.Context, tx *sql.Tx, id uuid.UUID) error {
	s := sqlbuilder.DeleteFrom("application_keys")
	s.Where(s.Equal("id", id))

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	_, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("executing delete: %w", err)
	}

	return nil
}
