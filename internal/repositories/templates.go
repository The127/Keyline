package repositories

import (
	"Keyline/utils"
	"context"

	"github.com/google/uuid"
)

type TemplateType string

const (
	EmailVerificationMailTemplate TemplateType = "email_verification"
)

type Template struct {
	BaseModel

	virtualServerId uuid.UUID
	fileId          uuid.UUID
	templateType    TemplateType
}

func NewTemplate(virtualServerId uuid.UUID, fileId uuid.UUID, templateType TemplateType) *Template {
	return &Template{
		BaseModel:       NewBaseModel(),
		virtualServerId: virtualServerId,
		fileId:          fileId,
		templateType:    templateType,
	}
}

func NewTemplateFromDB(base BaseModel, virtualServerId uuid.UUID, fileId uuid.UUID, templateType TemplateType) *Template {
	return &Template{
		BaseModel:       base,
		virtualServerId: virtualServerId,
		fileId:          fileId,
		templateType:    templateType,
	}
}

func (t *Template) VirtualServerId() uuid.UUID {
	return t.virtualServerId
}

func (t *Template) FileId() uuid.UUID {
	return t.fileId
}

func (t *Template) TemplateType() TemplateType {
	return t.templateType
}

type TemplateFilter struct {
	PagingInfo
	OrderInfo
	virtualServerId *uuid.UUID
	templateType    *TemplateType
	searchFilter    *SearchFilter
}

func NewTemplateFilter() *TemplateFilter {
	return &TemplateFilter{}
}

func (f *TemplateFilter) Clone() *TemplateFilter {
	clone := *f
	return &clone
}

func (f *TemplateFilter) VirtualServerId(virtualServerId uuid.UUID) *TemplateFilter {
	filter := f.Clone()
	filter.virtualServerId = &virtualServerId
	return filter
}

func (f *TemplateFilter) HasVirtualServerId() bool {
	return f.virtualServerId != nil
}

func (f *TemplateFilter) GetVirtualServerId() uuid.UUID {
	return utils.ZeroIfNil(f.virtualServerId)
}

func (f *TemplateFilter) TemplateType(templateType TemplateType) *TemplateFilter {
	filter := f.Clone()
	filter.templateType = &templateType
	return filter
}

func (f *TemplateFilter) HasTemplateType() bool {
	return f.templateType != nil
}

func (f *TemplateFilter) GetTemplateType() TemplateType {
	return utils.ZeroIfNil(f.templateType)
}

func (f *TemplateFilter) Search(searchFilter SearchFilter) *TemplateFilter {
	filter := f.Clone()
	filter.searchFilter = &searchFilter
	return filter
}

func (f *TemplateFilter) HasSearch() bool {
	return f.searchFilter != nil
}

func (f *TemplateFilter) GetSearch() SearchFilter {
	return *f.searchFilter
}

func (f *TemplateFilter) Pagination(page int, size int) *TemplateFilter {
	filter := f.Clone()
	filter.PagingInfo = PagingInfo{
		page: page,
		size: size,
	}
	return filter
}

func (f *TemplateFilter) HasPagination() bool {
	return !f.PagingInfo.IsZero()
}

func (f *TemplateFilter) GetPagingInfo() PagingInfo {
	return f.PagingInfo
}

func (f *TemplateFilter) Order(by string, direction string) *TemplateFilter {
	filter := f.Clone()
	filter.OrderInfo = OrderInfo{
		orderBy:  by,
		orderDir: direction,
	}
	return filter
}

func (f *TemplateFilter) HasOrder() bool {
	return !f.OrderInfo.IsZero()
}

func (f *TemplateFilter) GetOrderInfo() OrderInfo {
	return f.OrderInfo
}

//go:generate mockgen -destination=./mocks/template_repository.go -package=mocks Keyline/internal/repositories TemplateRepository
type TemplateRepository interface {
	Single(ctx context.Context, filter *TemplateFilter) (*Template, error)
	First(ctx context.Context, filter *TemplateFilter) (*Template, error)
	List(ctx context.Context, filter *TemplateFilter) ([]*Template, int, error)
	Insert(template *Template)
}
