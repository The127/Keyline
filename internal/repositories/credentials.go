package repositories

import (
	"Keyline/utils"
	"context"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
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

func (c *Credential) GetScanPointers() []any {
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

func (c *Credential) Details() any {
	return c.details
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

	result, ok := c.details.(*CredentialPasswordDetails)
	if ok {
		return result, nil
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

func (c *Credential) TotpDetails() (*CredentialTotpDetails, error) {
	if c._type != CredentialTypeTotp {
		return nil, fmt.Errorf("expected totp credential, got %s: %w", c._type, ErrWrongCredentialCast)
	}

	result, ok := c.details.(*CredentialTotpDetails)
	if ok {
		return result, nil
	}

	detailBytes, ok := c.details.([]byte)
	if !ok {
		return nil, fmt.Errorf("cannot access detail bytes: %w", ErrWrongCredentialCast)
	}

	totpDetails := CredentialTotpDetails{}
	err := json.Unmarshal(detailBytes, &totpDetails)
	if err != nil {
		return nil, fmt.Errorf("cannot unmarshal totp details: %w", err)
	}

	return &totpDetails, nil
}

func (c *Credential) ServiceUserKeyDetails() (*CredentialServiceUserKey, error) {
	if c._type != CredentialTypeServiceUserKey {
		return nil, fmt.Errorf("expected service user key credential, got %s: %w", c._type, ErrWrongCredentialCast)
	}

	result, ok := c.details.(*CredentialServiceUserKey)
	if ok {
		return result, nil
	}

	detailBytes, ok := c.details.([]byte)
	if !ok {
		return nil, fmt.Errorf("cannot access detail bytes: %w", ErrWrongCredentialCast)
	}

	serviceUserKeyDetails := CredentialServiceUserKey{}
	err := json.Unmarshal(detailBytes, &serviceUserKeyDetails)
	if err != nil {
		return nil, fmt.Errorf("cannot unmarshal service user key details: %w", err)
	}

	return &serviceUserKeyDetails, nil
}

// CredentialType represents a credential type.
// Use the following constants: CredentialTypePassword
type CredentialType string

const (
	CredentialTypePassword       CredentialType = "password"
	CredentialTypeTotp           CredentialType = "totp"
	CredentialTypeServiceUserKey CredentialType = "service_user_key"
)

type CredentialDetails interface {
	CredentialDetailType() CredentialType
}

type CredentialServiceUserKey struct {
	PublicKey string `json:"publicKey"`
}

func (d *CredentialServiceUserKey) CredentialDetailType() CredentialType {
	return CredentialTypeServiceUserKey
}

func (d *CredentialServiceUserKey) Value() (driver.Value, error) {
	return json.Marshal(d)
}

func (d *CredentialServiceUserKey) Scan(value any) error {
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("type assertion for credential failed")
	}

	return json.Unmarshal(bytes, &d)
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
	id     *uuid.UUID
	userId *uuid.UUID
	_type  *CredentialType
}

func NewCredentialFilter() CredentialFilter {
	return CredentialFilter{}
}

func (f CredentialFilter) Clone() CredentialFilter {
	return f
}

func (f CredentialFilter) Id(id uuid.UUID) CredentialFilter {
	filter := f.Clone()
	filter.id = &id
	return filter
}

func (f CredentialFilter) HasId() bool {
	return f.id != nil
}

func (f CredentialFilter) GetId() uuid.UUID {
	return utils.ZeroIfNil(f.id)
}

func (f CredentialFilter) UserId(userId uuid.UUID) CredentialFilter {
	filter := f.Clone()
	filter.userId = &userId
	return filter
}

func (f CredentialFilter) HasUserId() bool {
	return f.userId != nil
}

func (f CredentialFilter) GetUserId() uuid.UUID {
	return utils.ZeroIfNil(f.userId)
}

func (f CredentialFilter) Type(credentialType CredentialType) CredentialFilter {
	filter := f.Clone()
	filter._type = &credentialType
	return filter
}

func (f CredentialFilter) HasType() bool {
	return f._type != nil
}

func (f CredentialFilter) GetType() CredentialType {
	return utils.ZeroIfNil(f._type)
}

//go:generate mockgen -destination=./mocks/credential_repository.go -package=mocks Keyline/internal/repositories CredentialRepository
type CredentialRepository interface {
	Single(ctx context.Context, filter CredentialFilter) (*Credential, error)
	First(ctx context.Context, filter CredentialFilter) (*Credential, error)
	List(ctx context.Context, filter CredentialFilter) ([]*Credential, error)
	Insert(ctx context.Context, credential *Credential) error
	Update(ctx context.Context, credential *Credential) error
}
