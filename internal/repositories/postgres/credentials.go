package postgres

import (
	"Keyline/internal/change"
	"Keyline/internal/logging"
	"Keyline/internal/repositories"
	"Keyline/internal/repositories/postgres/pghelpers"
	"Keyline/utils"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/huandu/go-sqlbuilder"
)

type postgresCredential struct {
	postgresBaseModel
	userId  uuid.UUID
	type_   string
	details []byte
}

func mapCredential(credential *repositories.Credential) (*postgresCredential, error) {
	detailJson, err := json.Marshal(credential.Details())
	if err != nil {
		return nil, fmt.Errorf("failed to marshal details: %w", err)
	}

	return &postgresCredential{
		postgresBaseModel: mapBase(credential.BaseModel),
		userId:            credential.UserId(),
		type_:             string(credential.Type()),
		details:           detailJson,
	}, nil
}

func (c *postgresCredential) Map() (*repositories.Credential, error) {
	var details any
	credentialType := repositories.CredentialType(c.type_)

	switch credentialType {
	case repositories.CredentialTypeServiceUserKey:
		var serviceUserKey repositories.CredentialServiceUserKey
		err := json.Unmarshal(c.details, &serviceUserKey)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal service user key details: %w", err)
		}
		details = &serviceUserKey

	case repositories.CredentialTypePassword:
		var password repositories.CredentialPasswordDetails
		err := json.Unmarshal(c.details, &password)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal password details: %w", err)
		}
		details = &password

	case repositories.CredentialTypeTotp:
		var totp repositories.CredentialTotpDetails
		err := json.Unmarshal(c.details, &totp)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal totp details: %w", err)
		}
		details = &totp

	case repositories.CredentialTypeWebauthn:
		var webauthn repositories.CredentialWebauthnDetails
		err := json.Unmarshal(c.details, &webauthn)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal webauthn details: %w", err)
		}
		details = &webauthn

	default:
		return nil, fmt.Errorf("unsupported credential type: %s", c.type_)
	}

	return repositories.NewCredentialFromDB(
		c.MapBase(),
		c.userId,
		credentialType,
		details,
	), nil
}

func (c *postgresCredential) scan(row pghelpers.Row, additionalPtrs ...any) error {
	ptrs := []any{
		&c.id,
		&c.auditCreatedAt,
		&c.auditUpdatedAt,
		&c.xmin,
		&c.userId,
		&c.type_,
		&c.details,
	}

	ptrs = append(ptrs, additionalPtrs...)

	return row.Scan(ptrs...)
}

type CredentialRepository struct {
	db            *sql.DB
	changeTracker *change.Tracker
	entityType    int
}

func NewCredentialRepository(db *sql.DB, changeTracker *change.Tracker, entityType int) *CredentialRepository {
	return &CredentialRepository{
		db:            db,
		changeTracker: changeTracker,
		entityType:    entityType,
	}
}

func (r *CredentialRepository) selectQuery(filter *repositories.CredentialFilter) *sqlbuilder.SelectBuilder {
	s := sqlbuilder.Select(
		"id",
		"audit_created_at",
		"audit_updated_at",
		"xmin",
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

func (r *CredentialRepository) FirstOrErr(ctx context.Context, filter *repositories.CredentialFilter) (*repositories.Credential, error) {
	credential, err := r.FirstOrNil(ctx, filter)
	if err != nil {
		return nil, err
	}
	if credential == nil {
		return nil, utils.ErrCredentialNotFound
	}
	return credential, nil
}

func (r *CredentialRepository) FirstOrNil(ctx context.Context, filter *repositories.CredentialFilter) (*repositories.Credential, error) {
	s := r.selectQuery(filter)
	s.Limit(1)

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	row := r.db.QueryRowContext(ctx, query, args...)

	credential := &postgresCredential{}
	err := credential.scan(row)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, nil

	case err != nil:
		return nil, fmt.Errorf("scanning row: %w", err)
	}

	return credential.Map()
}

func (r *CredentialRepository) List(ctx context.Context, filter *repositories.CredentialFilter) ([]*repositories.Credential, error) {
	s := r.selectQuery(filter)

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying db: %w", err)
	}
	defer utils.PanicOnError(rows.Close, "closing rows")

	var credentials []*repositories.Credential
	for rows.Next() {
		credential := &postgresCredential{}
		err := credential.scan(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning row: %w", err)
		}

		mapped, err := credential.Map()
		if err != nil {
			return nil, fmt.Errorf("mapping credential: %w", err)
		}

		credentials = append(credentials, mapped)
	}

	return credentials, nil
}

func (r *CredentialRepository) Insert(credential *repositories.Credential) {
	r.changeTracker.Add(change.NewEntry(change.Added, r.entityType, credential))
}

func (r *CredentialRepository) ExecuteInsert(ctx context.Context, tx *sql.Tx, credential *repositories.Credential) error {
	mapped, err := mapCredential(credential)
	if err != nil {
		return fmt.Errorf("mapping credential: %w", err)
	}

	s := sqlbuilder.InsertInto("credentials").
		Cols(
			"id",
			"audit_created_at",
			"audit_updated_at",
			"user_id",
			"type",
			"details",
		).
		Values(
			mapped.id,
			mapped.auditCreatedAt,
			mapped.auditUpdatedAt,
			credential.UserId(),
			credential.Type(),
			credential.Details(),
		).
		Returning("xmin")

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	row := tx.QueryRowContext(ctx, query, args...)

	var xmin uint32
	err = row.Scan(&xmin)
	if err != nil {
		return fmt.Errorf("scanning row: %w", err)
	}

	credential.SetVersion(xmin)
	credential.ClearChanges()
	return nil
}

func (r *CredentialRepository) Update(credential *repositories.Credential) {
	r.changeTracker.Add(change.NewEntry(change.Updated, r.entityType, credential))
}

func (r *CredentialRepository) ExecuteUpdate(ctx context.Context, tx *sql.Tx, credential *repositories.Credential) error {
	if !credential.HasChanges() {
		return nil
	}

	mapped, err := mapCredential(credential)
	if err != nil {
		return fmt.Errorf("mapping credential: %w", err)
	}

	s := sqlbuilder.Update("credentials")
	s.Where(s.Equal("id", mapped.id))
	s.Where(s.Equal("xmin", mapped.xmin))

	for _, field := range credential.GetChanges() {
		switch field {
		case repositories.CredentialChangeType:
			s.SetMore(s.Assign("type", mapped.type_))

		case repositories.CredentialChangeDetails:
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
	err = row.Scan(&xmin)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return fmt.Errorf("updating credential: %w", repositories.ErrVersionMismatch)
	case err != nil:
		return fmt.Errorf("scanning row: %w", err)
	}

	credential.SetVersion(xmin)
	credential.ClearChanges()
	return nil
}

func (r *CredentialRepository) Delete(id uuid.UUID) {
	r.changeTracker.Add(change.NewEntry(change.Deleted, r.entityType, id))
}

func (r *CredentialRepository) ExecuteDelete(ctx context.Context, tx *sql.Tx, id uuid.UUID) error {
	s := sqlbuilder.DeleteFrom("credentials")
	s.Where(s.Equal("id", id))

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	_, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("executing sql: %w", err)
	}

	return nil
}
