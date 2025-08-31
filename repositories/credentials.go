package repositories

import (
	"Keyline/database"
	"Keyline/ioc"
	"Keyline/logging"
	"Keyline/middlewares"
	"Keyline/utils"
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/huandu/go-sqlbuilder"
	"github.com/pquerna/otp"
)

var (
	ErrWrongCredentialCast = errors.New("wrong credential cast")
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

func (c *Credential) getScanPointers() []any {
	return []any{
		&c.id,
		&c.auditCreatedAt,
		&c.auditUpdatedAt,
		&c.version,
		&c.userId,
		&c._type,
		&c.details,
	}
}

func (c *Credential) UserId() uuid.UUID {
	return c.userId
}

func (c *Credential) Type() CredentialType {
	return c._type
}

func (c *Credential) SetDetails(details CredentialDetails) {
	c._type = details.CredentialDetailType()
	c.details = details
	c.TrackChange("type", details.CredentialDetailType())
	c.TrackChange("details", details)
}

func (c *Credential) PasswordDetails() (*CredentialPasswordDetails, error) {
	if c._type != CredentialTypePassword {
		return nil, fmt.Errorf("expected password credential, got %s: %w", c._type, ErrWrongCredentialCast)
	}

	detailBytes, ok := c.details.([]byte)
	if !ok {
		return nil, fmt.Errorf("cannot access detail bytes: %w", ErrWrongCredentialCast)
	}

	passwordDetails := CredentialPasswordDetails{}
	err := json.Unmarshal(detailBytes, &passwordDetails)
	if err != nil {
		return nil, fmt.Errorf("cannot unmarshal password details: %w", err)
	}

	return &passwordDetails, nil
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
	_type  *CredentialType
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

func (f CredentialFilter) Type(credentialType CredentialType) CredentialFilter {
	filter := f.Clone()
	filter._type = &credentialType
	return filter
}

//go:generate mockgen -destination=./mocks/credential_repository.go -package=mocks Keyline/repositories CredentialRepository
type CredentialRepository interface {
	Single(ctx context.Context, filter CredentialFilter) (*Credential, error)
	First(ctx context.Context, filter CredentialFilter) (*Credential, error)
	List(ctx context.Context, filter CredentialFilter) ([]*Credential, error)
	Insert(ctx context.Context, credential *Credential) error
	Update(ctx context.Context, credential *Credential) error
}

type credentialRepository struct {
}

func NewCredentialRepository() CredentialRepository {
	return &credentialRepository{}
}

func (r *credentialRepository) selectQuery(filter CredentialFilter) *sqlbuilder.SelectBuilder {
	s := sqlbuilder.Select(
		"id",
		"audit_created_at",
		"audit_updated_at",
		"version",
		"user_id",
		"type",
		"details",
	).From("credentials")

	if filter.userId != nil {
		s.Where(s.Equal("user_id", filter.userId))
	}

	if filter._type != nil {
		s.Where(s.Equal("type", filter._type))
	}

	return s
}

func (r *credentialRepository) Single(ctx context.Context, filter CredentialFilter) (*Credential, error) {
	credential, err := r.First(ctx, filter)
	if err != nil {
		return nil, err
	}
	if credential == nil {
		return nil, utils.ErrCredentialNotFound
	}
	return credential, nil
}

func (r *credentialRepository) First(ctx context.Context, filter CredentialFilter) (*Credential, error) {
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

	credential := Credential{
		ModelBase: NewModelBase(),
	}
	err = row.Scan(credential.getScanPointers()...)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, nil

	case err != nil:
		return nil, fmt.Errorf("scanning row: %w", err)
	}

	return &credential, nil
}

func (r *credentialRepository) List(ctx context.Context, filter CredentialFilter) ([]*Credential, error) {
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

	var credentials []*Credential
	for rows.Next() {
		credential := Credential{
			ModelBase: NewModelBase(),
		}
		err = rows.Scan(credential.getScanPointers()...)
		if err != nil {
			return nil, fmt.Errorf("scanning row: %w", err)
		}
		credentials = append(credentials, &credential)
	}

	return credentials, nil
}

func (r *credentialRepository) Insert(ctx context.Context, credential *Credential) error {
	scope := middlewares.GetScope(ctx)
	dbService := ioc.GetDependency[database.DbService](scope)

	tx, err := dbService.GetTx()
	if err != nil {
		return fmt.Errorf("failed to open tx: %w", err)
	}

	s := sqlbuilder.InsertInto("credentials").
		Cols("user_id", "type", "details").
		Values(credential.userId, credential._type, credential.details).
		Returning("id", "audit_created_at", "audit_updated_at", "version")

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	row := tx.QueryRowContext(ctx, query, args...)

	err = row.Scan(&credential.id, &credential.auditCreatedAt, &credential.auditUpdatedAt, &credential.version)
	if err != nil {
		return fmt.Errorf("scanning row: %w", err)
	}

	credential.clearChanges()
	return nil
}

func (r *credentialRepository) Update(ctx context.Context, credential *Credential) error {
	scope := middlewares.GetScope(ctx)
	dbService := ioc.GetDependency[database.DbService](scope)

	tx, err := dbService.GetTx()
	if err != nil {
		return fmt.Errorf("failed to open tx: %w", err)
	}

	s := sqlbuilder.Update("credentials")
	for fieldName, value := range credential.changes {
		s.SetMore(s.Assign(fieldName, value))
	}
	s.SetMore(s.Assign("version", credential.version+1))

	s.Where(s.Equal("id", credential.id))
	s.Where(s.Equal("version", credential.version))
	s.Returning("audit_updated_at", "version")

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	row := tx.QueryRowContext(ctx, query, args...)

	err = row.Scan(&credential.auditUpdatedAt, &credential.version)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return fmt.Errorf("updating credential: %w", ErrVersionMismatch)
	case err != nil:
		return fmt.Errorf("scanning row: %w", err)
	}

	credential.clearChanges()
	return nil
}
