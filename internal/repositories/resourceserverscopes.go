package repositories

import (
	"Keyline/utils"
	"context"

	"github.com/google/uuid"
)

type ResourceServerScope struct {
	BaseModel
	virtualServerId  uuid.UUID
	projectId        uuid.UUID
	resourceServerId uuid.UUID

	scope       string
	name        string
	description string
}

func NewResourceServerScope(virtualServerId uuid.UUID, projectId uuid.UUID, resourceServerId uuid.UUID, scope string, name string) *ResourceServerScope {
	return &ResourceServerScope{
		BaseModel:        NewModelBase(),
		virtualServerId:  virtualServerId,
		projectId:        projectId,
		resourceServerId: resourceServerId,
		scope:            scope,
		name:             name,
	}
}

func (r *ResourceServerScope) GetScanPointers() []any {
	return []any{
		&r.id,
		&r.auditCreatedAt,
		&r.auditUpdatedAt,
		&r.version,
		&r.virtualServerId,
		&r.projectId,
		&r.resourceServerId,
		&r.scope,
		&r.name,
		&r.description,
	}
}

func (r *ResourceServerScope) Scope() string {
	return r.scope
}

func (r *ResourceServerScope) Name() string {
	return r.name
}

func (r *ResourceServerScope) SetName(name string) {
	r.TrackChange("name", name)
	r.name = name
}

func (r *ResourceServerScope) Description() string {
	return r.description
}

func (r *ResourceServerScope) SetDescription(description string) {
	r.description = description
	r.TrackChange("description", description)
}

func (r *ResourceServerScope) VirtualServerId() uuid.UUID {
	return r.virtualServerId
}

func (r *ResourceServerScope) ProjectId() uuid.UUID {
	return r.projectId
}

func (r *ResourceServerScope) ResourceServerId() uuid.UUID {
	return r.resourceServerId
}

type ResourceServerScopeFilter struct {
	PagingInfo
	OrderInfo
	virtualServerId  *uuid.UUID
	projectId        *uuid.UUID
	resourceServerId *uuid.UUID
	id               *uuid.UUID
	searchFilter     *SearchFilter
}

func NewResourceServerScopeFilter() ResourceServerScopeFilter {
	return ResourceServerScopeFilter{}
}

func (f ResourceServerScopeFilter) Clone() ResourceServerScopeFilter {
	return f
}

func (f ResourceServerScopeFilter) VirtualServerId(virtualServerId uuid.UUID) ResourceServerScopeFilter {
	filter := f.Clone()
	filter.virtualServerId = &virtualServerId
	return filter
}

func (f ResourceServerScopeFilter) HasVirtualServerId() bool {
	return f.virtualServerId != nil
}

func (f ResourceServerScopeFilter) GetVirtualServerId() uuid.UUID {
	return utils.ZeroIfNil(f.virtualServerId)
}

func (f ResourceServerScopeFilter) ProjectId(projectId uuid.UUID) ResourceServerScopeFilter {
	filter := f.Clone()
	filter.projectId = &projectId
	return filter
}

func (f ResourceServerScopeFilter) HasProjectId() bool {
	return f.projectId != nil
}

func (f ResourceServerScopeFilter) GetProjectId() uuid.UUID {
	return utils.ZeroIfNil(f.projectId)
}

func (f ResourceServerScopeFilter) ResourceServerId(resourceServerId uuid.UUID) ResourceServerScopeFilter {
	filter := f.Clone()
	filter.resourceServerId = &resourceServerId
	return filter
}

func (f ResourceServerScopeFilter) HasResourceServerId() bool {
	return f.resourceServerId != nil
}

func (f ResourceServerScopeFilter) GetResourceServerId() uuid.UUID {
	return utils.ZeroIfNil(f.resourceServerId)
}

func (f ResourceServerScopeFilter) Id(id uuid.UUID) ResourceServerScopeFilter {
	filter := f.Clone()
	filter.id = &id
	return filter
}

func (f ResourceServerScopeFilter) HasId() bool {
	return f.id != nil
}

func (f ResourceServerScopeFilter) GetId() uuid.UUID {
	return utils.ZeroIfNil(f.id)
}

func (f ResourceServerScopeFilter) Search(searchFilter SearchFilter) ResourceServerScopeFilter {
	filter := f.Clone()
	filter.searchFilter = &searchFilter
	return filter
}

func (f ResourceServerScopeFilter) HasSearch() bool {
	return f.searchFilter != nil
}

func (f ResourceServerScopeFilter) GetSearch() SearchFilter {
	return *f.searchFilter
}

func (f ResourceServerScopeFilter) Pagination(page int, size int) ResourceServerScopeFilter {
	filter := f.Clone()
	filter.PagingInfo = PagingInfo{
		page: page,
		size: size,
	}
	return filter
}

func (f ResourceServerScopeFilter) HasPagination() bool {
	return !f.PagingInfo.IsZero()
}

func (f ResourceServerScopeFilter) GetPagingInfo() PagingInfo {
	return f.PagingInfo
}

func (f ResourceServerScopeFilter) Order(by string, direction string) ResourceServerScopeFilter {
	filter := f.Clone()
	filter.OrderInfo = OrderInfo{
		orderBy:  by,
		orderDir: direction,
	}
	return filter
}

func (f ResourceServerScopeFilter) HasOrder() bool {
	return !f.OrderInfo.IsZero()
}

func (f ResourceServerScopeFilter) GetOrderInfo() OrderInfo {
	return f.OrderInfo
}

//go:generate mockgen -destination=./mocks/resource_server_scope_repository.go -package=mocks Keyline/internal/repositories ResourceServerScopeRepository
type ResourceServerScopeRepository interface {
	List(ctx context.Context, filter ResourceServerScopeFilter) ([]*ResourceServerScope, int, error)
	Single(ctx context.Context, filter ResourceServerScopeFilter) (*ResourceServerScope, error)
	First(ctx context.Context, filter ResourceServerScopeFilter) (*ResourceServerScope, error)
	Insert(ctx context.Context, resourceServerScope *ResourceServerScope) error
	Update(ctx context.Context, resourceServerScope *ResourceServerScope) error
	Delete(ctx context.Context, id uuid.UUID) error
}
