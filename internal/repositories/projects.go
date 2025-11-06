package repositories

import (
	"Keyline/utils"
	"context"

	"github.com/google/uuid"
)

type Project struct {
	ModelBase

	virtualServerId uuid.UUID

	slug        string
	name        string
	description string

	systemProject bool
}

func NewProject(virtualServerId uuid.UUID, slug string, name string, description string) *Project {
	return &Project{
		ModelBase:       NewModelBase(),
		virtualServerId: virtualServerId,
		slug:            slug,
		name:            name,
		description:     description,
	}
}

func NewSystemProject(virtualServerId uuid.UUID) *Project {
	project := NewProject(virtualServerId, "system", "System Project", "Keyline internal project for system internal resources.")
	project.systemProject = true
	return project
}

func (p *Project) GetScanPointers() []any {
	return []any{
		&p.id,
		&p.auditCreatedAt,
		&p.auditUpdatedAt,
		&p.version,
		&p.virtualServerId,
		&p.slug,
		&p.name,
		&p.description,
		&p.systemProject,
	}
}

func (p *Project) Slug() string {
	return p.slug
}

func (p *Project) SystemProject() bool {
	return p.systemProject
}

func (p *Project) Description() string {
	return p.description
}

func (p *Project) SetDescription(description string) {
	p.description = description
	p.TrackChange("description", description)
}

func (p *Project) Name() string {
	return p.name
}

func (p *Project) SetName(name string) {
	p.name = name
	p.TrackChange("name", name)
}

func (p *Project) VirtualServerId() uuid.UUID {
	return p.virtualServerId
}

type ProjectFilter struct {
	PagingInfo
	OrderInfo
	virtualServerId *uuid.UUID
	slug            *string
	id              *uuid.UUID
	searchFilter    *SearchFilter
}

func NewProjectFilter() ProjectFilter {
	return ProjectFilter{}
}

func (f ProjectFilter) Clone() ProjectFilter {
	return f
}

func (f ProjectFilter) VirtualServerId(virtualServerId uuid.UUID) ProjectFilter {
	filter := f.Clone()
	filter.virtualServerId = &virtualServerId
	return filter
}

func (f ProjectFilter) HasVirtualServerId() bool {
	return f.virtualServerId != nil
}

func (f ProjectFilter) GetVirtualServerId() uuid.UUID {
	return utils.ZeroIfNil(f.virtualServerId)
}

func (f ProjectFilter) Slug(slug string) ProjectFilter {
	filter := f.Clone()
	filter.slug = &slug
	return filter
}

func (f ProjectFilter) HasSlug() bool {
	return f.slug != nil
}

func (f ProjectFilter) GetSlug() string {
	return utils.ZeroIfNil(f.slug)
}

func (f ProjectFilter) Id(id uuid.UUID) ProjectFilter {
	filter := f.Clone()
	filter.id = &id
	return filter
}

func (f ProjectFilter) HasId() bool {
	return f.id != nil
}

func (f ProjectFilter) GetId() uuid.UUID {
	return utils.ZeroIfNil(f.id)
}

func (f ProjectFilter) Search(searchFilter SearchFilter) ProjectFilter {
	filter := f.Clone()
	filter.searchFilter = &searchFilter
	return filter
}

func (f ProjectFilter) HasSearch() bool {
	return f.searchFilter != nil
}

func (f ProjectFilter) GetSearch() SearchFilter {
	return *f.searchFilter
}

func (f ProjectFilter) Pagination(page int, size int) ProjectFilter {
	filter := f.Clone()
	filter.PagingInfo = PagingInfo{
		page: page,
		size: size,
	}
	return filter
}

func (f ProjectFilter) HasPagination() bool {
	return !f.PagingInfo.IsZero()
}

func (f ProjectFilter) GetPagingInfo() PagingInfo {
	return f.PagingInfo
}

func (f ProjectFilter) Order(by string, direction string) ProjectFilter {
	filter := f.Clone()
	filter.OrderInfo = OrderInfo{
		orderBy:  by,
		orderDir: direction,
	}
	return filter
}

func (f ProjectFilter) HasOrder() bool {
	return !f.OrderInfo.IsZero()
}

func (f ProjectFilter) GetOrderInfo() OrderInfo {
	return f.OrderInfo
}

//go:generate mockgen -destination=./mocks/project_repository.go -package=mocks Keyline/internal/repositories ProjectRepository
type ProjectRepository interface {
	List(ctx context.Context, filter ProjectFilter) ([]*Project, int, error)
	Single(ctx context.Context, filter ProjectFilter) (*Project, error)
	First(ctx context.Context, filter ProjectFilter) (*Project, error)
	Insert(ctx context.Context, project *Project) error
	Update(ctx context.Context, project *Project) error
	Delete(ctx context.Context, id uuid.UUID) error
}
