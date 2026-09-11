package repositories

import (
	"context"
	"github.com/The127/Keyline/internal/change"
	"github.com/The127/Keyline/utils"

	"github.com/google/uuid"
)

type ApplicationKeyChange int

type ApplicationKey struct {
	BaseModel
	change.List[ApplicationKeyChange]

	applicationId uuid.UUID
	kid           string
	publicKey     string
}

func NewApplicationKey(applicationId uuid.UUID, kid string, publicKey string) *ApplicationKey {
	return &ApplicationKey{
		BaseModel:     NewBaseModel(),
		List:          change.NewChanges[ApplicationKeyChange](),
		applicationId: applicationId,
		kid:           kid,
		publicKey:     publicKey,
	}
}

func NewApplicationKeyFromDB(base BaseModel, applicationId uuid.UUID, kid string, publicKey string) *ApplicationKey {
	return &ApplicationKey{
		BaseModel:     base,
		List:          change.NewChanges[ApplicationKeyChange](),
		applicationId: applicationId,
		kid:           kid,
		publicKey:     publicKey,
	}
}

func (k *ApplicationKey) ApplicationId() uuid.UUID {
	return k.applicationId
}

func (k *ApplicationKey) Kid() string {
	return k.kid
}

func (k *ApplicationKey) PublicKey() string {
	return k.publicKey
}

type ApplicationKeyFilter struct {
	id            *uuid.UUID
	applicationId *uuid.UUID
	kid           *string
}

func NewApplicationKeyFilter() *ApplicationKeyFilter {
	return &ApplicationKeyFilter{}
}

func (f *ApplicationKeyFilter) Clone() *ApplicationKeyFilter {
	clone := *f
	return &clone
}

func (f *ApplicationKeyFilter) Id(id uuid.UUID) *ApplicationKeyFilter {
	filter := f.Clone()
	filter.id = &id
	return filter
}

func (f *ApplicationKeyFilter) HasId() bool {
	return f.id != nil
}

func (f *ApplicationKeyFilter) GetId() uuid.UUID {
	return utils.ZeroIfNil(f.id)
}

func (f *ApplicationKeyFilter) ApplicationId(applicationId uuid.UUID) *ApplicationKeyFilter {
	filter := f.Clone()
	filter.applicationId = &applicationId
	return filter
}

func (f *ApplicationKeyFilter) HasApplicationId() bool {
	return f.applicationId != nil
}

func (f *ApplicationKeyFilter) GetApplicationId() uuid.UUID {
	return utils.ZeroIfNil(f.applicationId)
}

func (f *ApplicationKeyFilter) Kid(kid string) *ApplicationKeyFilter {
	filter := f.Clone()
	filter.kid = &kid
	return filter
}

func (f *ApplicationKeyFilter) HasKid() bool {
	return f.kid != nil
}

func (f *ApplicationKeyFilter) GetKid() string {
	return utils.ZeroIfNil(f.kid)
}

//go:generate mockgen -destination=./mocks/application_key_repository.go -package=mocks Keyline/internal/repositories ApplicationKeyRepository
type ApplicationKeyRepository interface {
	FirstOrErr(ctx context.Context, filter *ApplicationKeyFilter) (*ApplicationKey, error)
	FirstOrNil(ctx context.Context, filter *ApplicationKeyFilter) (*ApplicationKey, error)
	List(ctx context.Context, filter *ApplicationKeyFilter) ([]*ApplicationKey, error)
	Insert(applicationKey *ApplicationKey)
	Delete(id uuid.UUID)
}
