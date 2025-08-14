package repositories

import (
	"Keyline/ioc"
	"Keyline/logging"
	"Keyline/middlewares"
	"Keyline/services"
	"context"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"time"
)

type Credential struct {
	id uuid.UUID

	auditCreatedAt time.Time
	auditUpdatedAt time.Time

	userId uuid.UUID

	_type   CredentialType
	details any
}

func NewCredential(userId uuid.UUID, details CredentialDetails) *Credential {
	return &Credential{
		userId:  userId,
		_type:   details.CredentialDetailType(),
		details: details,
	}
}

// CredentialType represents a credential type.
// Use the following constants: CredentialTypePassword
type CredentialType string

const (
	CredentialTypePassword CredentialType = "password"
)

type CredentialDetails interface {
	CredentialDetailType() CredentialType
}

type CredentialPasswordDetails struct {
	HashedPassword string `json:"hashedPassword"`
	Temporary      bool   `json:"temporary"`
}

func (d *CredentialPasswordDetails) CredentialDetailType() CredentialType {
	return CredentialTypePassword
}

func (d *CredentialPasswordDetails) Value() (driver.Value, error) {
	return json.Marshal(d)
}

func (d *CredentialPasswordDetails) Scan(value any) error {
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("type assertion for credential failed")
	}

	return json.Unmarshal(bytes, &d)
}

type CredentialFilter struct {
	userId *uuid.UUID
}

func (f CredentialFilter) Clone() CredentialFilter {
	return f
}

func (f CredentialFilter) UserId(userId uuid.UUID) CredentialFilter {
	filter := f.Clone()
	filter.userId = &userId
	return filter
}

type CredentialRepository struct {
}

func (r *CredentialRepository) Insert(ctx context.Context, credential *Credential) error {
	scope := middlewares.GetScope(ctx)
	dbService := ioc.GetDependency[*services.DbService](scope)

	tx, err := dbService.GetTx()
	if err != nil {
		return fmt.Errorf("failed to open tx: %w", err)
	}

	s := `
insert into credentials
	(user_id, type, details)
values ($1, $2, $3)
returning id, audit_created_at, audit_updated_at`

	logging.Logger.Debug("sql: %s", s)
	row := tx.QueryRow(s, credential.userId, credential._type, credential.details)

	err = row.Scan(&credential.id, &credential.auditCreatedAt, &credential.auditUpdatedAt)
	if err != nil {
		return fmt.Errorf("scanning row: %w", err)
	}

	return nil
}
