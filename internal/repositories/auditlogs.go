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
		ModelBase:       NewModelBase(),
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
		ModelBase:       NewModelBase(),
		virtualServerId: virtualServerId,
		userId:          &userId,
		requestType:     request.GetRequestName(),
		request:         string(requestJsonMap),
		allowed:         false,
	}, nil
}

func (a *AuditLog) getScanPointers() []any {
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

func (f AuditLogFilter) Pagination(page int, pageSize int) AuditLogFilter {
	filter := f.Clone()
	filter.pagingInfo = pagingInfo{
		page: page,
		size: pageSize,
	}
	return filter
}

func (f AuditLogFilter) Order(by string, direction string) AuditLogFilter {
	filter := f.Clone()
	filter.orderInfo = orderInfo{
		orderBy:  by,
		orderDir: direction,
	}
	return filter
}

type AuditLogRepository interface {
	List(ctx context.Context, filter AuditLogFilter) ([]*AuditLog, int, error)
	Insert(ctx context.Context, auditLog *AuditLog) error
}

type auditLogRepository struct{}

func NewAuditLogRepository() AuditLogRepository {
	return &auditLogRepository{}
}

func (r *auditLogRepository) selectQuery(filter AuditLogFilter) *sqlbuilder.SelectBuilder {
	s := sqlbuilder.Select(
		"id",
		"audit_created_at",
		"audit_updated_at",
		"version",
		"virtual_server_id",
		"user_id",
		"request_type",
		"request",
		"response",
		"allowed",
		"allow_reason_type",
		"allow_reason",
	).From("audit_logs")

	if filter.virtualServerId != nil {
		s.Where(s.Equal("virtual_server_id", filter.virtualServerId))
	}

	if filter.userId != nil {
		s.Where(s.Equal("user_id", filter.userId))
	}

	filter.pagingInfo.apply(s)
	filter.orderInfo.apply(s)

	return s
}

func (r *auditLogRepository) List(ctx context.Context, filter AuditLogFilter) ([]*AuditLog, int, error) {
	scope := middlewares.GetScope(ctx)
	dbService := ioc.GetDependency[database.DbService](scope)

	tx, err := dbService.GetTx()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to open tx: %w", err)
	}

	s := r.selectQuery(filter)
	s.SelectMore("count(*) over()")

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	rows, err := tx.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("querying db: %w", err)
	}
	defer utils.PanicOnError(rows.Close, "closing rows")

	var auditLogs []*AuditLog
	var totalCount int
	for rows.Next() {
		auditLog := AuditLog{
			ModelBase: NewModelBase(),
		}
		err = rows.Scan(append(auditLog.getScanPointers(), &totalCount)...)
		if err != nil {
			return nil, 0, fmt.Errorf("scanning row: %w", err)
		}

		auditLogs = append(auditLogs, &auditLog)
	}

	return auditLogs, totalCount, nil
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
