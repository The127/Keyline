package repositories

import (
	"Keyline/utils"
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

type AuditLog struct {
	BaseModel

	virtualServerId uuid.UUID
	userId          *uuid.UUID

	requestType string
	request     string
	response    *string

	allowed         bool
	allowReasonType *string
	allowReason     *string
}

type Request interface {
	GetRequestName() string
}

type AllowReason interface {
	GetReasonType() string
}

func NewAllowedAuditLog(virtualServerId uuid.UUID, userId uuid.UUID, request Request, response any, allowReason AllowReason) (*AuditLog, error) {
	requestJsonBytes, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	responseJsonBytes, err := json.Marshal(response)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}
	responseJsonString := string(responseJsonBytes)

	allowReasonJsonBytes, err := json.Marshal(allowReason)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal allow reason: %w", err)
	}
	allowReasonJsonString := string(allowReasonJsonBytes)

	return &AuditLog{
		BaseModel:       NewBaseModel(),
		virtualServerId: virtualServerId,
		userId:          &userId,
		requestType:     request.GetRequestName(),
		request:         string(requestJsonBytes),
		response:        &responseJsonString,
		allowed:         true,
		allowReasonType: utils.Ptr(allowReason.GetReasonType()),
		allowReason:     &allowReasonJsonString,
	}, nil
}

func NewDeniedAuditLog(virtualServerId uuid.UUID, userId uuid.UUID, request Request) (*AuditLog, error) {
	requestJsonMap, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	return &AuditLog{
		BaseModel:       NewBaseModel(),
		virtualServerId: virtualServerId,
		userId:          &userId,
		requestType:     request.GetRequestName(),
		request:         string(requestJsonMap),
		allowed:         false,
	}, nil
}

func (a *AuditLog) GetScanPointers() []any {
	return []any{
		&a.id,
		&a.auditCreatedAt,
		&a.auditUpdatedAt,
		&a.version,
		&a.virtualServerId,
		&a.userId,
		&a.requestType,
		&a.request,
		&a.response,
		&a.allowed,
		&a.allowReasonType,
		&a.allowReason,
	}
}

func (a *AuditLog) VirtualServerId() uuid.UUID {
	return a.virtualServerId
}

func (a *AuditLog) UserId() *uuid.UUID {
	return a.userId
}

func (a *AuditLog) RequestType() string {
	return a.requestType
}

func (a *AuditLog) Request() string {
	return a.request
}

func (a *AuditLog) Response() *string {
	return a.response
}

func (a *AuditLog) Allowed() bool {
	return a.allowed
}

func (a *AuditLog) AllowReasonType() *string {
	return a.allowReasonType
}

func (a *AuditLog) AllowReason() *string {
	return a.allowReason
}

type AuditLogFilter struct {
	PagingInfo
	OrderInfo
	virtualServerId *uuid.UUID
	userId          *uuid.UUID
}

func NewAuditLogFilter() AuditLogFilter {
	return AuditLogFilter{}
}

func (f AuditLogFilter) Clone() AuditLogFilter {
	return f
}

func (f AuditLogFilter) VirtualServerId(virtualServerId uuid.UUID) AuditLogFilter {
	filter := f.Clone()
	filter.virtualServerId = &virtualServerId
	return filter
}

func (f AuditLogFilter) HasVirtualServerId() bool {
	return f.virtualServerId != nil
}

func (f AuditLogFilter) GetVirtualServerId() uuid.UUID {
	return utils.ZeroIfNil(f.virtualServerId)
}

func (f AuditLogFilter) UserId(userId uuid.UUID) AuditLogFilter {
	filter := f.Clone()
	filter.userId = &userId
	return filter
}

func (f AuditLogFilter) HasUserId() bool {
	return f.userId != nil
}

func (f AuditLogFilter) GetUserId() uuid.UUID {
	return utils.ZeroIfNil(f.userId)
}

func (f AuditLogFilter) Pagination(page int, pageSize int) AuditLogFilter {
	filter := f.Clone()
	filter.PagingInfo = PagingInfo{
		page: page,
		size: pageSize,
	}
	return filter
}

func (f AuditLogFilter) HasPagination() bool {
	return !f.PagingInfo.IsZero()
}

func (f AuditLogFilter) GetPagingInfo() PagingInfo {
	return f.PagingInfo
}

func (f AuditLogFilter) Order(by string, direction string) AuditLogFilter {
	filter := f.Clone()
	filter.OrderInfo = OrderInfo{
		orderBy:  by,
		orderDir: direction,
	}
	return filter
}

func (f AuditLogFilter) HasOrder() bool {
	return !f.OrderInfo.IsZero()
}

func (f AuditLogFilter) GetOrderInfo() OrderInfo {
	return f.OrderInfo
}

//go:generate mockgen -destination=./mocks/auditlog_repository.go -package=mocks Keyline/internal/repositories AuditLogRepository
type AuditLogRepository interface {
	List(ctx context.Context, filter AuditLogFilter) ([]*AuditLog, int, error)
	Insert(auditLog *AuditLog)
}
