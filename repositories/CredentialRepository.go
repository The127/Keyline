package repositories

import (
	"Keyline/database"
	"Keyline/ioc"
	"Keyline/logging"
	"Keyline/middlewares"
	"context"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/huandu/go-sqlbuilder"
	"github.com/pquerna/otp"
)

type Credential struct {
	ModelBase

	userId uuid.UUID

	_type   CredentialType
	details any
}

func NewCredential(userId uuid.UUID, details CredentialDetails) *Credential {
	return &Credential{
		ModelBase: NewModelBase(),
		userId:    userId,
		_type:     details.CredentialDetailType(),
		details:   details,
	}
}

// CredentialType represents a credential type.
// Use the following constants: CredentialTypePassword
type CredentialType string

const (
	CredentialTypePassword CredentialType = "password"
	CredentialTypeTotp     CredentialType = "totp"
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

type CredentialTotpDetails struct {
	Issuer      string        `json:"issuer"`
	AccountName string        `json:"accountName"`
	Period      uint          `json:"period"`
	SecretSize  uint          `json:"secretSize"`
	Secret      []byte        `json:"secret"`
	Digits      otp.Digits    `json:"digits"`
	Algorithm   otp.Algorithm `json:"algorithm"`
}

func (d *CredentialTotpDetails) CredentialDetailType() CredentialType {
	return CredentialTypeTotp
}

func (d *CredentialTotpDetails) Value() (driver.Value, error) {
	return json.Marshal(d)
}

func (d *CredentialTotpDetails) Scan(value any) error {
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("type assertion for credential failed")
	}

	return json.Unmarshal(bytes, &d)
}

type CredentialFilter struct {
	userId *uuid.UUID
}

func NewCredentialFilter() CredentialFilter {
	return CredentialFilter{}
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
	dbService := ioc.GetDependency[*database.DbService](scope)

	tx, err := dbService.GetTx()
	if err != nil {
		return fmt.Errorf("failed to open tx: %w", err)
	}

	s := sqlbuilder.InsertInto("credentials").
		Cols("user_id", "type", "details").
		Values(credential.userId, credential._type, credential.details).
		Returning("id", "audit_created_at", "audit_updated_at")

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	row := tx.QueryRowContext(ctx, query, args...)

	err = row.Scan(&credential.id, &credential.auditCreatedAt, &credential.auditUpdatedAt)
	if err != nil {
		return fmt.Errorf("scanning row: %w", err)
	}

	credential.ClearChanges()
	return nil
}
