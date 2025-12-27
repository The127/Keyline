package repositories

import (
	"Keyline/internal/change"
	"Keyline/utils"
	"context"

	"github.com/google/uuid"
)

type ResourceServerChange int

const (
	ResourceServerChangeName ResourceServerChange = iota
	ResourceServerChangeDescription
)

type ResourceServer struct {
	BaseModel
	change.List[ResourceServerChange]

	virtualServerId uuid.UUID
	projectId       uuid.UUID

	slug        string
	name        string
	description string
}

func NewResourceServer(virtualServerId uuid.UUID, projectId uuid.UUID, slug string, name string, description string) *ResourceServer {
	return &ResourceServer{
		BaseModel:       NewBaseModel(),
		List:            change.NewChanges[ResourceServerChange](),
		virtualServerId: virtualServerId,
		projectId:       projectId,
		slug:            slug,
		name:            name,
		description:     description,
	}
}

func NewResourceServerFromDB(base BaseModel, virtualServerId uuid.UUID, projectId uuid.UUID, slug string, name string, description string) *ResourceServer {
	return &ResourceServer{
		BaseModel:       base,
		List:            change.NewChanges[ResourceServerChange](),
		virtualServerId: virtualServerId,
		projectId:       projectId,
		slug:            slug,
		name:            name,
		description:     description,
	}
}

func (r *ResourceServer) Slug() string {
	return r.slug
}

func (r *ResourceServer) Name() string {
	return r.name
}

func (r *ResourceServer) SetName(name string) {
	if r.name == name {
		return
	}

	r.name = name
	r.TrackChange(ResourceServerChangeName)
}

func (r *ResourceServer) Description() string {
	return r.description
}

func (r *ResourceServer) SetDescription(description string) {
	if r.description == description {
		return
	}

	r.description = description
	r.TrackChange(ResourceServerChangeDescription)
}

func (r *ResourceServer) VirtualServerId() uuid.UUID {
	return r.virtualServerId
}

func (r *ResourceServer) ProjectId() uuid.UUID {
	return r.projectId
}

type ResourceServerFilter struct {
	PagingInfo
	OrderInfo
	virtualServerId *uuid.UUID
	projectId       *uuid.UUID
	id              *uuid.UUID
	slug            *string
	searchFilter    *SearchFilter
}

func NewResourceServerFilter() *ResourceServerFilter {
	return &ResourceServerFilter{}
}

func (f *ResourceServerFilter) Clone() *ResourceServerFilter {
	clone := *f
	return &clone
}

func (f *ResourceServerFilter) Id(id uuid.UUID) *ResourceServerFilter {
	filter := f.Clone()
	filter.id = &id
	return filter
}

func (f *ResourceServerFilter) HasId() bool {
	return f.id != nil
}

func (f *ResourceServerFilter) GetId() uuid.UUID {
	return utils.ZeroIfNil(f.id)
}

func (f *ResourceServerFilter) Slug(slug string) *ResourceServerFilter {
	filter := f.Clone()
	filter.slug = &slug
	return filter
}

func (f *ResourceServerFilter) HasSlug() bool {
	return f.slug != nil
}

func (f *ResourceServerFilter) GetSlug() string {
	return utils.ZeroIfNil(f.slug)
}

func (f *ResourceServerFilter) VirtualServerId(virtualServerId uuid.UUID) *ResourceServerFilter {
	filter := f.Clone()
	filter.virtualServerId = &virtualServerId
	return filter
}

func (f *ResourceServerFilter) HasVirtualServerId() bool {
	return f.virtualServerId != nil
}

func (f *ResourceServerFilter) GetVirtualServerId() uuid.UUID {
	return utils.ZeroIfNil(f.virtualServerId)
}

func (f *ResourceServerFilter) ProjectId(projectId uuid.UUID) *ResourceServerFilter {
	filter := f.Clone()
	filter.projectId = &projectId
	return filter
}

func (f *ResourceServerFilter) HasProjectId() bool {
	return f.projectId != nil
}

func (f *ResourceServerFilter) GetProjectId() uuid.UUID {
	return utils.ZeroIfNil(f.projectId)
}

func (f *ResourceServerFilter) Search(searchFilter SearchFilter) *ResourceServerFilter {
	filter := f.Clone()
	filter.searchFilter = &searchFilter
	return filter
}

func (f *ResourceServerFilter) HasSearch() bool {
	return f.searchFilter != nil
}

func (f *ResourceServerFilter) GetSearch() SearchFilter {
	return *f.searchFilter
}

func (f *ResourceServerFilter) Pagination(page int, size int) *ResourceServerFilter {
	filter := f.Clone()
	filter.PagingInfo = PagingInfo{
		page: page,
		size: size,
	}
	return filter
}

func (f *ResourceServerFilter) HasPagination() bool {
	return !f.PagingInfo.IsZero()
}

func (f *ResourceServerFilter) GetPagingInfo() PagingInfo {
	return f.PagingInfo
}

func (f *ResourceServerFilter) Order(by string, direction string) *ResourceServerFilter {
	filter := f.Clone()
	filter.OrderInfo = OrderInfo{
		orderBy:  by,
		orderDir: direction,
	}
	return filter
}

func (f *ResourceServerFilter) HasOrder() bool {
	return !f.OrderInfo.IsZero()
}

func (f *ResourceServerFilter) GetOrderInfo() OrderInfo {
	return f.OrderInfo
}

//go:generate mockgen -destination=./mocks/resource_server_repository.go -package=mocks Keyline/internal/repositories ResourceServerRepository
type ResourceServerRepository interface {
	FirstOrErr(ctx context.Context, filter *ResourceServerFilter) (*ResourceServer, error)
	FirstOrNil(ctx context.Context, filter *ResourceServerFilter) (*ResourceServer, error)
	List(ctx context.Context, filter *ResourceServerFilter) ([]*ResourceServer, int, error)
	Insert(resourceServer *ResourceServer)
	Update(resourceServer *ResourceServer)
	Delete(id uuid.UUID)
}
