package repositories

import (
	"Keyline/internal/change"
	"Keyline/utils"
	"context"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

var (
	ErrWrongCredentialCast = errors.New("wrong credential cast")
)

type CredentialChange int

const (
	CredentialChangeDetails CredentialChange = iota
	CredentialChangeType
)

type Credential struct {
	BaseModel
	change.List[CredentialChange]

	userId uuid.UUID

	_type   CredentialType
	details any
}

func NewCredential(userId uuid.UUID, details CredentialDetails) *Credential {
	return &Credential{
		BaseModel: NewBaseModel(),
		List:      change.NewChanges[CredentialChange](),
		userId:    userId,
		_type:     details.CredentialDetailType(),
		details:   details,
	}
}

func NewCredentialFromDB(base BaseModel) *Credential {
	return &Credential{
		BaseModel: base,
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

	c.TrackChange(CredentialChangeDetails)
	c.TrackChange(CredentialChangeType)
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

func (c *Credential) WebauthnDetails() (*CredentialWebauthnDetails, error) {
	if c._type != CredentialTypeWebauthn {
		return nil, fmt.Errorf("expected webauthn credential, got %s: %w", c._type, ErrWrongCredentialCast)
	}

	result, ok := c.details.(*CredentialWebauthnDetails)
	if ok {
		return result, nil
	}

	detailBytes, ok := c.details.([]byte)
	if !ok {
		return nil, fmt.Errorf("cannot access detail bytes: %w", ErrWrongCredentialCast)
	}

	webauthnDetails := CredentialWebauthnDetails{}
	err := json.Unmarshal(detailBytes, &webauthnDetails)
	if err != nil {
		return nil, fmt.Errorf("cannot unmarshal webauthn details: %w", err)
	}

	return &webauthnDetails, nil
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
	CredentialTypeWebauthn       CredentialType = "webauthn"
)

type CredentialDetails interface {
	CredentialDetailType() CredentialType
}

type CredentialServiceUserKey struct {
	Kid       string `json:"kid"`
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

type CredentialWebauthnDetails struct {
	CredentialId       string `json:"credentialId"`
	PublicKeyAlgorithm int    `json:"publicKeyAlgorithm"`
	PublicKey          []byte `json:"publicKey"`
}

func (d *CredentialWebauthnDetails) CredentialDetailType() CredentialType {
	return CredentialTypeWebauthn
}

func (d *CredentialWebauthnDetails) Value() (driver.Value, error) {
	return json.Marshal(d)
}

func (d *CredentialWebauthnDetails) Scan(value any) error {
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("type assertion for credential failed")
	}

	return json.Unmarshal(bytes, &d)
}

type CredentialTotpDetails struct {
	Secret    string `json:"secret"`
	Digits    int    `json:"digits"`
	Algorithm int    `json:"algorithm"`
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
	id              *uuid.UUID
	userId          *uuid.UUID
	_type           *CredentialType
	detailId        *string
	detailKid       *string
	detailPublicKey *string
}

func NewCredentialFilter() *CredentialFilter {
	return &CredentialFilter{}
}

func (f *CredentialFilter) Clone() *CredentialFilter {
	clone := *f
	return &clone
}

func (f *CredentialFilter) Id(id uuid.UUID) *CredentialFilter {
	filter := f.Clone()
	filter.id = &id
	return filter
}

func (f *CredentialFilter) HasId() bool {
	return f.id != nil
}

func (f *CredentialFilter) GetId() uuid.UUID {
	return utils.ZeroIfNil(f.id)
}

func (f *CredentialFilter) UserId(userId uuid.UUID) *CredentialFilter {
	filter := f.Clone()
	filter.userId = &userId
	return filter
}

func (f *CredentialFilter) HasUserId() bool {
	return f.userId != nil
}

func (f *CredentialFilter) GetUserId() uuid.UUID {
	return utils.ZeroIfNil(f.userId)
}

func (f *CredentialFilter) DetailPublicKey(publicKey string) *CredentialFilter {
	filter := f.Clone()
	filter.detailPublicKey = &publicKey
	return filter
}

func (f *CredentialFilter) HasDetailPublicKey() bool {
	return f.detailPublicKey != nil
}

func (f *CredentialFilter) GetDetailPublicKey() string {
	return utils.ZeroIfNil(f.detailPublicKey)
}

func (f *CredentialFilter) Type(credentialType CredentialType) *CredentialFilter {
	filter := f.Clone()
	filter._type = &credentialType
	return filter
}

func (f *CredentialFilter) HasType() bool {
	return f._type != nil
}

func (f *CredentialFilter) GetType() CredentialType {
	return utils.ZeroIfNil(f._type)
}

func (f *CredentialFilter) DetailKid(key string) *CredentialFilter {
	filter := f.Clone()
	filter.detailKid = &key
	return filter
}

func (f *CredentialFilter) HasDetailKid() bool {
	return f.detailKid != nil
}

func (f *CredentialFilter) GetDetailKid() string {
	return utils.ZeroIfNil(f.detailKid)
}

func (f *CredentialFilter) DetailsId(id string) *CredentialFilter {
	filter := f.Clone()
	filter.detailId = &id
	return filter
}

func (f *CredentialFilter) HasDetailsId() bool {
	return f.detailId != nil
}

func (f *CredentialFilter) GetDetailsId() string {
	return utils.ZeroIfNil(f.detailId)
}

//go:generate mockgen -destination=./mocks/credential_repository.go -package=mocks Keyline/internal/repositories CredentialRepository
type CredentialRepository interface {
	Single(ctx context.Context, filter *CredentialFilter) (*Credential, error)
	FirstOrNil(ctx context.Context, filter *CredentialFilter) (*Credential, error)
	List(ctx context.Context, filter *CredentialFilter) ([]*Credential, error)
	Insert(credential *Credential)
	Update(credential *Credential)
	Delete(id uuid.UUID)
}
