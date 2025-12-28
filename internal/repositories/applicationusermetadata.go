package repositories

import (
	"Keyline/internal/change"
	"Keyline/utils"
	"context"

	"github.com/google/uuid"
)

type ApplicationUserMetadataChange int

const (
	ApplicationUserMetadataChangeMetadata ApplicationUserMetadataChange = iota
)

type ApplicationUserMetadata struct {
	BaseModel
	change.List[ApplicationUserMetadataChange]

	applicationId uuid.UUID
	userId        uuid.UUID

	metadata string
}

func NewApplicationUserMetadata(applicationId uuid.UUID, userId uuid.UUID, metadata string) *ApplicationUserMetadata {
	return &ApplicationUserMetadata{
		BaseModel:     NewBaseModel(),
		List:          change.NewChanges[ApplicationUserMetadataChange](),
		applicationId: applicationId,
		userId:        userId,
		metadata:      metadata,
	}
}

func NewApplicationUserMetadataFromDB(
	base BaseModel,
	applicationId uuid.UUID,
	userId uuid.UUID,
	metadata string,
) *ApplicationUserMetadata {
	return &ApplicationUserMetadata{
		BaseModel:     base,
		applicationId: applicationId,
		userId:        userId,
		metadata:      metadata,
	}
}

func (a *ApplicationUserMetadata) ApplicationId() uuid.UUID {
	return a.applicationId
}

func (a *ApplicationUserMetadata) UserId() uuid.UUID {
	return a.userId
}

func (a *ApplicationUserMetadata) Metadata() string {
	return a.metadata
}

func (a *ApplicationUserMetadata) SetMetadata(metadata string) {
	if a.metadata == metadata {
		return
	}

	a.metadata = metadata
	a.TrackChange(ApplicationUserMetadataChangeMetadata)
}

type ApplicationUserMetadataFilter struct {
	applicationId  *uuid.UUID
	applicationIds *[]uuid.UUID
	userId         *uuid.UUID
}

func NewApplicationUserMetadataFilter() *ApplicationUserMetadataFilter {
	return &ApplicationUserMetadataFilter{}
}

func (f *ApplicationUserMetadataFilter) Clone() *ApplicationUserMetadataFilter {
	clone := *f
	return &clone
}

func (f *ApplicationUserMetadataFilter) ApplicationId(applicationId uuid.UUID) *ApplicationUserMetadataFilter {
	filter := f.Clone()
	filter.applicationId = &applicationId
	return filter
}

func (f *ApplicationUserMetadataFilter) HasApplicationId() bool {
	return f.applicationId != nil
}

func (f *ApplicationUserMetadataFilter) GetApplicationId() uuid.UUID {
	return utils.ZeroIfNil(f.applicationId)
}

func (f *ApplicationUserMetadataFilter) ApplicationIds(applicationIds []uuid.UUID) *ApplicationUserMetadataFilter {
	filter := f.Clone()
	filter.applicationIds = &applicationIds
	return filter
}

func (f *ApplicationUserMetadataFilter) HasApplicationIds() bool {
	return f.applicationIds != nil
}

func (f *ApplicationUserMetadataFilter) GetApplicationIds() []uuid.UUID {
	if f.applicationIds == nil {
		return []uuid.UUID{}
	}
	return *f.applicationIds
}

func (f *ApplicationUserMetadataFilter) UserId(userId uuid.UUID) *ApplicationUserMetadataFilter {
	filter := f.Clone()
	filter.userId = &userId
	return filter
}

func (f *ApplicationUserMetadataFilter) HasUserId() bool {
	return f.userId != nil
}

func (f *ApplicationUserMetadataFilter) GetUserId() uuid.UUID {
	return utils.ZeroIfNil(f.userId)
}

//go:generate mockgen -destination=./mocks/application_user_metadata_repository.go -package=mocks Keyline/internal/repositories ApplicationUserMetadataRepository
type ApplicationUserMetadataRepository interface {
	FirstOrErr(ctx context.Context, filter *ApplicationUserMetadataFilter) (*ApplicationUserMetadata, error)
	FirstOrNil(ctx context.Context, filter *ApplicationUserMetadataFilter) (*ApplicationUserMetadata, error)
	List(ctx context.Context, filter *ApplicationUserMetadataFilter) ([]*ApplicationUserMetadata, int, error)
	Insert(applicationUserMetadata *ApplicationUserMetadata)
	Update(applicationUserMetadata *ApplicationUserMetadata)
}
