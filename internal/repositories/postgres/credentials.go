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

type CredentialRepository struct {
	db            *sql.DB
	changeTracker *change.Tracker
	entityType    int
}

func NewCredentialRepository(db *sql.DB, changeTracker change.Tracker, entityType int) repositories.CredentialRepository {
	return &CredentialRepository{
		db:            db,
		changeTracker: &changeTracker,
		entityType:    entityType,
	}
}

func (r *CredentialRepository) selectQuery(filter repositories.CredentialFilter) *sqlbuilder.SelectBuilder {
	s := sqlbuilder.Select(
		"id",
		"audit_created_at",
		"audit_updated_at",
		"version",
		"user_id",
		"type",
		"details",
	).From("credentials")

	if filter.HasId() {
		s.Where(s.Equal("id", filter.GetId()))
	}

	if filter.HasUserId() {
		s.Where(s.Equal("user_id", filter.GetUserId()))
	}

	if filter.HasType() {
		s.Where(s.Equal("type", filter.GetType()))
	}

	if filter.HasDetailsId() {
		s.Where(s.Equal("details->>'credentialId'", filter.GetDetailsId()))
	}

	if filter.HasDetailPublicKey() {
		s.Where(s.Equal("details->>'publicKey'", filter.GetDetailPublicKey()))
	}

	if filter.HasDetailKid() {
		s.Where(s.Equal("details->>'kid'", filter.GetDetailKid()))
	}

	return s
}

func (r *CredentialRepository) Single(ctx context.Context, filter repositories.CredentialFilter) (*repositories.Credential, error) {
	credential, err := r.First(ctx, filter)
	if err != nil {
		return nil, err
	}
	if credential == nil {
		return nil, utils.ErrCredentialNotFound
	}
	return credential, nil
}

func (r *CredentialRepository) First(ctx context.Context, filter repositories.CredentialFilter) (*repositories.Credential, error) {
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

	credential := repositories.Credential{
		BaseModel: repositories.NewModelBase(),
	}
	err = row.Scan(credential.GetScanPointers()...)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, nil

	case err != nil:
		return nil, fmt.Errorf("scanning row: %w", err)
	}

	return &credential, nil
}

func (r *CredentialRepository) List(ctx context.Context, filter repositories.CredentialFilter) ([]*repositories.Credential, error) {
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

	var credentials []*repositories.Credential
	for rows.Next() {
		credential := repositories.Credential{
			BaseModel: repositories.NewModelBase(),
		}
		err = rows.Scan(credential.GetScanPointers()...)
		if err != nil {
			return nil, fmt.Errorf("scanning row: %w", err)
		}
		credentials = append(credentials, &credential)
	}

	return credentials, nil
}

func (r *CredentialRepository) Insert(ctx context.Context, credential *repositories.Credential) error {
	scope := middlewares.GetScope(ctx)
	dbService := ioc.GetDependency[database.DbService](scope)

	tx, err := dbService.GetTx()
	if err != nil {
		return fmt.Errorf("failed to open tx: %w", err)
	}

	s := sqlbuilder.InsertInto("credentials").
		Cols("user_id", "type", "details").
		Values(credential.UserId(), credential.Type(), credential.Details()).
		Returning("id", "audit_created_at", "audit_updated_at", "version")

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	row := tx.QueryRowContext(ctx, query, args...)

	err = row.Scan(credential.InsertPointers()...)
	if err != nil {
		return fmt.Errorf("scanning row: %w", err)
	}

	credential.ClearChanges()
	return nil
}

func (r *CredentialRepository) Update(ctx context.Context, credential *repositories.Credential) error {
	scope := middlewares.GetScope(ctx)
	dbService := ioc.GetDependency[database.DbService](scope)

	tx, err := dbService.GetTx()
	if err != nil {
		return fmt.Errorf("failed to open tx: %w", err)
	}

	s := sqlbuilder.Update("credentials")
	for fieldName, value := range credential.Changes() {
		s.SetMore(s.Assign(fieldName, value))
	}
	s.SetMore(s.Assign("version", credential.Version()+1))

	s.Where(s.Equal("id", credential.Id()))
	s.Where(s.Equal("version", credential.Version()))
	s.Returning("audit_updated_at", "version")

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	row := tx.QueryRowContext(ctx, query, args...)

	err = row.Scan(credential.UpdatePointers()...)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return fmt.Errorf("updating credential: %w", repositories.ErrVersionMismatch)
	case err != nil:
		return fmt.Errorf("scanning row: %w", err)
	}

	credential.ClearChanges()
	return nil
}

func (r *CredentialRepository) Delete(ctx context.Context, id uuid.UUID) error {
	scope := middlewares.GetScope(ctx)
	dbService := ioc.GetDependency[database.DbService](scope)

	tx, err := dbService.GetTx()
	if err != nil {
		return fmt.Errorf("failed to open tx: %w", err)
	}

	s := sqlbuilder.DeleteFrom("credentials")
	s.Where(s.Equal("id", id))

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	_, err = tx.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("executing sql: %w", err)
	}

	return nil
}
