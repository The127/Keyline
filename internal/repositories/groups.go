package repositories

import (
	"Keyline/internal/change"
	"Keyline/utils"
	"context"

	"github.com/google/uuid"
)

type GroupChange int

const (
	GroupChangeName GroupChange = iota
	GroupChangeDescription
)

type Group struct {
	BaseModel
	change.List[GroupChange]

	virtualServerId uuid.UUID

	name        string
	description string
}

func NewGroup(virtualServerId uuid.UUID, name string, description string) *Group {
	return &Group{
		BaseModel:       NewBaseModel(),
		List:            change.NewChanges[GroupChange](),
		virtualServerId: virtualServerId,
		name:            name,
		description:     description,
	}
}

func (g *Group) GetScanPointers() []any {
	return []any{
		&g.id,
		&g.auditCreatedAt,
		&g.auditUpdatedAt,
		&g.version,
		&g.virtualServerId,
		&g.name,
		&g.description,
	}
}

func (g *Group) Name() string {
	return g.name
}

func (g *Group) SetName(name string) {
	if g.name == name {
		return
	}

	g.name = name
	g.TrackChange(GroupChangeName)
}

func (g *Group) Description() string {
	return g.description
}

func (g *Group) SetDescription(description string) {
	if g.description == description {
		return
	}

	g.description = description
	g.TrackChange(GroupChangeDescription)
}

func (g *Group) VirtualServerId() uuid.UUID {
	return g.virtualServerId
}

type GroupFilter struct {
	PagingInfo
	OrderInfo
	name            *string
	virtualServerId *uuid.UUID
	id              *uuid.UUID
	searchFilter    *SearchFilter
}

func NewGroupFilter() GroupFilter {
	return GroupFilter{}
}

func (f GroupFilter) Clone() GroupFilter {
	return f
}

func (f GroupFilter) Pagination(page int, size int) GroupFilter {
	filter := f.Clone()
	filter.PagingInfo = PagingInfo{
		page: page,
		size: size,
	}
	return filter
}

func (f GroupFilter) HasPagination() bool {
	return !f.PagingInfo.IsZero()
}

func (f GroupFilter) GetPagingInfo() PagingInfo {
	return f.PagingInfo
}

func (f GroupFilter) Order(by string, direction string) GroupFilter {
	filter := f.Clone()
	filter.OrderInfo = OrderInfo{
		orderBy:  by,
		orderDir: direction,
	}
	return filter
}

func (f GroupFilter) HasOrder() bool {
	return !f.OrderInfo.IsZero()
}

func (f GroupFilter) GetOrderInfo() OrderInfo {
	return f.OrderInfo
}

func (f GroupFilter) Search(searchFilter SearchFilter) GroupFilter {
	filter := f.Clone()
	filter.searchFilter = &searchFilter
	return filter
}

func (f GroupFilter) HasSearch() bool {
	return f.searchFilter != nil
}

func (f GroupFilter) GetSearch() SearchFilter {
	return *f.searchFilter
}

func (f GroupFilter) Name(name string) GroupFilter {
	filter := f.Clone()
	filter.name = &name
	return filter
}

func (f GroupFilter) HasName() bool {
	return f.name != nil
}

func (f GroupFilter) GetName() string {
	return utils.ZeroIfNil(f.name)
}

func (f GroupFilter) VirtualServerId(virtualServerId uuid.UUID) GroupFilter {
	filter := f.Clone()
	filter.virtualServerId = &virtualServerId
	return filter
}

func (f GroupFilter) HasVirtualServerId() bool {
	return f.virtualServerId != nil
}

func (f GroupFilter) GetVirtualServerId() uuid.UUID {
	return utils.ZeroIfNil(f.virtualServerId)
}

func (f GroupFilter) Id(id uuid.UUID) GroupFilter {
	filter := f.Clone()
	filter.id = &id
	return filter
}

func (f GroupFilter) HasId() bool {
	return f.id != nil
}

func (f GroupFilter) GetId() uuid.UUID {
	return utils.ZeroIfNil(f.id)
}

//go:generate mockgen -destination=./mocks/group_repository.go -package=mocks Keyline/internal/repositories GroupRepository
type GroupRepository interface {
	Single(ctx context.Context, filter GroupFilter) (*Group, error)
	First(ctx context.Context, filter GroupFilter) (*Group, error)
	List(ctx context.Context, filter GroupFilter) ([]*Group, int, error)
	Insert(group *Group)
	Update(group *Group)
	Delete(id uuid.UUID)
}
