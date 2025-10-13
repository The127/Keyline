package repositories

import (
	"Keyline/internal/database"
	"Keyline/internal/logging"
	"Keyline/internal/middlewares"
	"Keyline/ioc"
	"Keyline/utils"
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/huandu/go-sqlbuilder"
)

type AuditLog struct {
	ModelBase

	virtualServerId uuid.UUID
	userId          *uuid.UUID

	requestType string
	request     map[string]any
	response    *map[string]any

	allowed         bool
	allowReasonType *string
	allowReason     *map[string]any
}

type Request interface {
	GetRequestName() string
}

type AllowReason interface {
	GetReasonType() string
}

func NewAllowedAuditLog(virtualServerId uuid.UUID, userId uuid.UUID, request Request, response any, allowReason AllowReason) (*AuditLog, error) {
	requestJsonMap, err := toJsonMap(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	responseJsonMap, err := toJsonMap(response)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}

	allowReasonJsonMap, err := toJsonMap(allowReason)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal allow reason: %w", err)
	}

	return &AuditLog{
		ModelBase:       NewModelBase(),
		virtualServerId: virtualServerId,
		userId:          &userId,
		requestType:     request.GetRequestName(),
		request:         requestJsonMap,
		response:        &responseJsonMap,
		allowed:         true,
		allowReasonType: utils.Ptr(allowReason.GetReasonType()),
		allowReason:     &allowReasonJsonMap,
	}, nil
}

func NewDeniedAuditLog(virtualServerId uuid.UUID, userId uuid.UUID, request Request) (*AuditLog, error) {
	requestJsonMap, err := toJsonMap(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	return &AuditLog{
		ModelBase:       NewModelBase(),
		virtualServerId: virtualServerId,
		userId:          &userId,
		requestType:     request.GetRequestName(),
		request:         requestJsonMap,
		allowed:         false,
	}, nil
}

func toJsonMap(request any) (map[string]any, error) {
	jsonBytes, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal: %w", err)
	}

	requestJsonMap := map[string]any{}
	err = json.Unmarshal(jsonBytes, &requestJsonMap)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal: %w", err)
	}

	return requestJsonMap, nil
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

func (a *AuditLog) Request() map[string]any {
	return a.request
}

func (a *AuditLog) Response() *map[string]any {
	return a.response
}

func (a *AuditLog) Allowed() bool {
	return a.allowed
}

func (a *AuditLog) AllowReasonType() *string {
	return a.allowReasonType
}

func (a *AuditLog) AllowReason() *map[string]any {
	return a.allowReason
}

type AuditLogFilter struct {
	pagingInfo
	orderInfo
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

func (f AuditLogFilter) UserId(userId uuid.UUID) AuditLogFilter {
	filter := f.Clone()
	filter.userId = &userId
	return filter
}

type AuditLogRepository interface {
	Insert(ctx context.Context, auditLog *AuditLog) error
}

type auditLogRepository struct{}

func NewAuditLogRepository() AuditLogRepository {
	return &auditLogRepository{}
}

func (r *auditLogRepository) Insert(ctx context.Context, auditLog *AuditLog) error {
	scope := middlewares.GetScope(ctx)
	dbService := ioc.GetDependency[database.DbService](scope)

	tx, err := dbService.GetTx()
	if err != nil {
		return fmt.Errorf("failed to open tx: %w", err)
	}

	s := sqlbuilder.InsertInto("audit_logs").
		Cols("virtual_server_id", "user_id", "request_type", "request", "response", "allowed", "allow_reason_type", "allow_reason").
		Values(
			auditLog.virtualServerId,
			auditLog.userId,
			auditLog.requestType,
			auditLog.request,
			auditLog.response,
			auditLog.allowed,
			auditLog.allowReasonType,
			auditLog.allowReason,
		).Returning("id", "audit_created_at", "audit_updated_at", "version")

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	row := tx.QueryRowContext(ctx, query, args...)

	err = row.Scan(&auditLog.id, &auditLog.auditCreatedAt, &auditLog.auditUpdatedAt, &auditLog.version)
	if err != nil {
		return fmt.Errorf("scanning row: %w", err)
	}

	auditLog.clearChanges()
	return nil
}
