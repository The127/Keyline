package repositories

import (
	"Keyline/utils"
	"context"
	"time"

	"github.com/google/uuid"
)

type Role struct {
	ModelBase

	virtualServerId uuid.UUID
	applicationId   uuid.UUID

	name        string
	description string

	requireMfa  bool
	maxTokenAge *time.Duration
}

func NewApplicationRole(virtualServerId uuid.UUID, applicationId uuid.UUID, name string, description string) *Role {
	return &Role{
		ModelBase:       NewModelBase(),
		virtualServerId: virtualServerId,
		applicationId:   applicationId,
		name:            name,
		description:     description,
	}
}

func (r *Role) GetScanPointers() []any {
	return []any{
		&r.id,
		&r.auditCreatedAt,
		&r.auditUpdatedAt,
		&r.version,
		&r.virtualServerId,
		&r.applicationId,
		&r.name,
		&r.description,
		&r.requireMfa,
		&r.maxTokenAge,
	}
}

func (r *Role) Name() string {
	return r.name
}

func (r *Role) SetName(name string) {
	r.TrackChange("name", name)
	r.name = name
}

func (r *Role) Description() string {
	return r.description
}

func (r *Role) SetDescription(description string) {
	r.TrackChange("description", description)
	r.description = description
}

func (r *Role) VirtualServerId() uuid.UUID {
	return r.virtualServerId
}

func (r *Role) ApplicationId() uuid.UUID {
	return r.applicationId
}

func (r *Role) RequireMfa() bool {
	return r.requireMfa
}

func (r *Role) SetRequireMfa(requireMfa bool) {
	r.TrackChange("require_mfa", requireMfa)
	r.requireMfa = requireMfa
}

func (r *Role) MaxTokenAge() *time.Duration {
	return r.maxTokenAge
}

func (r *Role) SetMaxTokenAge(maxTokenAge *time.Duration) {
	r.TrackChange("max_token_age", maxTokenAge)
	r.maxTokenAge = maxTokenAge
}

type RoleFilter struct {
	PagingInfo
	OrderInfo
	name            *string
	id              *uuid.UUID
	virtualServerId *uuid.UUID
	applicationId   *uuid.UUID
	searchFilter    *SearchFilter
}

func NewRoleFilter() RoleFilter {
	return RoleFilter{}
}

func (f RoleFilter) Clone() RoleFilter {
	return f
}

func (f RoleFilter) Name(name string) RoleFilter {
	filter := f.Clone()
	filter.name = &name
	return filter
}

func (f RoleFilter) HasName() bool {
	return f.name != nil
}

func (f RoleFilter) GetName() string {
	return utils.ZeroIfNil(f.name)
}

func (f RoleFilter) Id(id uuid.UUID) RoleFilter {
	filter := f.Clone()
	filter.id = &id
	return filter
}

func (f RoleFilter) HasId() bool {
	return f.id != nil
}

func (f RoleFilter) GetId() uuid.UUID {
	return utils.ZeroIfNil(f.id)
}

func (f RoleFilter) ApplicationId(applicationId uuid.UUID) RoleFilter {
	filter := f.Clone()
	filter.applicationId = &applicationId
	return filter
}

func (f RoleFilter) HasApplicationId() bool {
	return f.applicationId != nil
}

func (f RoleFilter) GetApplicationId() uuid.UUID {
	return utils.ZeroIfNil(f.applicationId)
}

func (f RoleFilter) Search(searchFilter SearchFilter) RoleFilter {
	filter := f.Clone()
	filter.searchFilter = &searchFilter
	return filter
}

func (f RoleFilter) HasSearch() bool {
	return f.searchFilter != nil
}

func (f RoleFilter) GetSearch() SearchFilter {
	return *f.searchFilter
}

func (f RoleFilter) Pagination(page int, size int) RoleFilter {
	filter := f.Clone()
	filter.PagingInfo = PagingInfo{
		page: page,
		size: size,
	}
	return filter
}

func (f RoleFilter) HasPagination() bool {
	return !f.PagingInfo.IsZero()
}

func (f RoleFilter) GetPagingInfo() PagingInfo {
	return f.PagingInfo
}

func (f RoleFilter) Order(by string, direction string) RoleFilter {
	filter := f.Clone()
	filter.OrderInfo = OrderInfo{
		orderBy:  by,
		orderDir: direction,
	}
	return filter
}

func (f RoleFilter) HasOrder() bool {
	return !f.OrderInfo.IsZero()
}

func (f RoleFilter) GetOrderInfo() OrderInfo {
	return f.OrderInfo
}

func (f RoleFilter) VirtualServerId(virtualServerId uuid.UUID) RoleFilter {
	filter := f.Clone()
	filter.virtualServerId = &virtualServerId
	return filter
}

func (f RoleFilter) HasVirtualServerId() bool {
	return f.virtualServerId != nil
}

func (f RoleFilter) GetVirtualServerId() uuid.UUID {
	return utils.ZeroIfNil(f.virtualServerId)
}

//go:generate mockgen -destination=./mocks/role_repository.go -package=mocks Keyline/internal/repositories RoleRepository
type RoleRepository interface {
	List(ctx context.Context, filter RoleFilter) ([]*Role, int, error)
	Single(ctx context.Context, filter RoleFilter) (*Role, error)
	First(ctx context.Context, filter RoleFilter) (*Role, error)
	Insert(ctx context.Context, role *Role) error
}
