package repositories

import (
	"Keyline/internal/change"
	"Keyline/utils"
	"context"

	"github.com/google/uuid"
)

type UserChange int

const (
	UserChangeDisplayName UserChange = iota
	UserChangeEmailVerified
	UserChangeMetadata
)

type User struct {
	BaseModel
	change.List[UserChange]

	virtualServerId uuid.UUID

	username    string
	displayName string

	primaryEmail  string
	emailVerified bool

	serviceUser bool

	metadata string
}

func NewUser(username string, displayName string, primaryEmail string, virtualServerId uuid.UUID) *User {
	return &User{
		BaseModel:       NewBaseModel(),
		List:            change.NewChanges[UserChange](),
		virtualServerId: virtualServerId,
		username:        username,
		displayName:     displayName,
		primaryEmail:    primaryEmail,
		serviceUser:     false,
		metadata:        "{}",
	}
}

func NewSystemUser(username string) *User {
	return &User{
		BaseModel:       NewBaseModel(),
		List:            change.NewChanges[UserChange](),
		virtualServerId: uuid.Nil,
		username:        username,
		displayName:     username,
		serviceUser:     true,
		metadata:        "{}",
	}
}

func NewServiceUser(username string, virtualServerId uuid.UUID) *User {
	return &User{
		BaseModel:       NewBaseModel(),
		List:            change.NewChanges[UserChange](),
		virtualServerId: virtualServerId,
		username:        username,
		displayName:     username,
		serviceUser:     true,
		metadata:        "{}",
	}
}

func NewUserFromDB(base BaseModel, virtualServerId uuid.UUID, username string, displayName string, primaryEmail string, emailVerified bool, serviceUser bool, metadata string) *User {
	return &User{
		BaseModel:       base,
		List:            change.NewChanges[UserChange](),
		virtualServerId: virtualServerId,
		username:        username,
		displayName:     displayName,
		primaryEmail:    primaryEmail,
		emailVerified:   emailVerified,
		serviceUser:     serviceUser,
		metadata:        metadata,
	}
}

func (m *User) VirtualServerId() uuid.UUID {
	return m.virtualServerId
}

func (m *User) Username() string {
	return m.username
}

func (m *User) DisplayName() string {
	return m.displayName
}

func (m *User) SetDisplayName(displayName string) {
	if m.displayName == displayName {
		return
	}

	m.displayName = displayName
	m.TrackChange(UserChangeDisplayName)
}

func (m *User) IsServiceUser() bool {
	return m.serviceUser
}

func (m *User) PrimaryEmail() string {
	return m.primaryEmail
}

func (m *User) EmailVerified() bool {
	return m.emailVerified
}

func (m *User) SetEmailVerified(emailVerified bool) {
	if m.emailVerified == emailVerified {
		return
	}

	m.emailVerified = emailVerified
	m.TrackChange(UserChangeEmailVerified)
}

func (m *User) Metadata() string {
	return m.metadata
}

func (m *User) SetMetadata(metadata string) {
	if m.metadata == metadata {
		return
	}

	m.metadata = metadata
	m.TrackChange(UserChangeMetadata)
}

type UserFilter struct {
	PagingInfo
	OrderInfo
	virtualServerId *uuid.UUID
	id              *uuid.UUID
	username        *string
	serviceUser     *bool
	searchFilter    *SearchFilter
	includeMetadata bool
}

func NewUserFilter() UserFilter {
	return UserFilter{}
}

func (f UserFilter) Clone() UserFilter {
	return f
}

func (f UserFilter) VirtualServerId(virtualServerId uuid.UUID) UserFilter {
	filter := f.Clone()
	filter.virtualServerId = &virtualServerId
	return filter
}

func (f UserFilter) HasVirtualServerId() bool {
	return f.virtualServerId != nil
}

func (f UserFilter) GetVirtualServerId() uuid.UUID {
	return utils.ZeroIfNil(f.virtualServerId)
}

func (f UserFilter) Id(id uuid.UUID) UserFilter {
	filter := f.Clone()
	filter.id = &id
	return filter
}

func (f UserFilter) HasId() bool {
	return f.id != nil
}

func (f UserFilter) GetId() uuid.UUID {
	return utils.ZeroIfNil(f.id)
}

func (f UserFilter) ServiceUser(serviceUser bool) UserFilter {
	filter := f.Clone()
	filter.serviceUser = &serviceUser
	return filter
}

func (f UserFilter) HasServiceUser() bool {
	return f.serviceUser != nil
}

func (f UserFilter) GetServiceUser() bool {
	return utils.ZeroIfNil(f.serviceUser)
}

func (f UserFilter) Username(username string) UserFilter {
	filter := f.Clone()
	filter.username = &username
	return filter
}

func (f UserFilter) HasUsername() bool {
	return f.username != nil
}

func (f UserFilter) GetUsername() string {
	return utils.ZeroIfNil(f.username)
}

func (f UserFilter) IncludeMetadata() UserFilter {
	filter := f.Clone()
	filter.includeMetadata = true
	return filter
}

func (f UserFilter) GetIncludeMetadata() bool {
	return f.includeMetadata
}

func (f UserFilter) Pagination(page int, size int) UserFilter {
	filter := f.Clone()
	filter.PagingInfo = PagingInfo{
		page: page,
		size: size,
	}
	return filter
}

func (f UserFilter) HasPagination() bool {
	return !f.PagingInfo.IsZero()
}

func (f UserFilter) GetPagingInfo() PagingInfo {
	return f.PagingInfo
}

func (f UserFilter) Order(by string, direction string) UserFilter {
	filter := f.Clone()
	filter.OrderInfo = OrderInfo{
		orderBy:  by,
		orderDir: direction,
	}
	return filter
}

func (f UserFilter) HasOrder() bool {
	return !f.OrderInfo.IsZero()
}

func (f UserFilter) GetOrderInfo() OrderInfo {
	return f.OrderInfo
}

func (f UserFilter) Search(searchFilter SearchFilter) UserFilter {
	filter := f.Clone()
	filter.searchFilter = &searchFilter
	return filter
}

func (f UserFilter) HasSearch() bool {
	return f.searchFilter != nil
}

func (f UserFilter) GetSearch() SearchFilter {
	return *f.searchFilter
}

//go:generate mockgen -destination=./mocks/user_repository.go -package=mocks Keyline/internal/repositories UserRepository
type UserRepository interface {
	List(ctx context.Context, filter UserFilter) ([]*User, int, error)
	Single(ctx context.Context, filter UserFilter) (*User, error)
	First(ctx context.Context, filter UserFilter) (*User, error)
	Insert(user *User)
	Update(user *User)
}
